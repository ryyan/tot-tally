// handlers.go contains the logic for processing HTTP requests and rendering templates.
package web

import (
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	totConfig "tot-tally/internal/config"
	totCore "tot-tally/internal/core"
	totModels "tot-tally/internal/models"
	totShards "tot-tally/internal/shards"
	totStats "tot-tally/internal/stats"
	totStorage "tot-tally/internal/storage"
)

// Server handles all HTTP requests and routes.
type Server struct {
	config        *totConfig.Config
	core          *totCore.Service
	store         *totStorage.Repository
	stats         *totStats.Engine
	shards        *totShards.Pool
	templateIndex *template.Template
	templateTot   *template.Template
}

// NewServer initializes the HTTP router with its dependencies.
func NewServer(cfg *totConfig.Config, c *totCore.Service, s *totStorage.Repository, e *totStats.Engine, p *totShards.Pool) *Server {
	// Try to find templates. In tests, they might be in a different relative path.
	paths := []string{"assets/", "../../assets/", "../assets/"}
	var indexPath, totPath string
	for _, p := range paths {
		if _, err := os.Stat(p + "index.html"); err == nil {
			indexPath = p + "index.html"
			totPath = p + "tot.html"
			break
		}
	}

	if indexPath == "" {
		// Fallback to original paths if not found in common test locations
		indexPath = "assets/index.html"
		totPath = "assets/tot.html"
	}

	return &Server{
		config:        cfg,
		core:          c,
		store:         s,
		stats:         e,
		shards:        p,
		templateIndex: template.Must(template.ParseFiles(indexPath)),
		templateTot:   template.Must(template.ParseFiles(totPath)),
	}
}

func (s *Server) homeHandler(w http.ResponseWriter, req *http.Request) (string, error) {
	var flashKey string
	if cookie, err := req.Cookie("flash_msg"); err == nil {
		flashKey = cookie.Value
		// Consume the flash message immediately.
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	}

	msg := totConfig.FlashMessages[flashKey]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err := s.templateIndex.Execute(w, totModels.HomePageData{
		FlashMessage: msg,
		IsErrorFlash: strings.HasPrefix(msg, "Error:"),
	})
	return "", err
}

func (s *Server) getTotHandler(w http.ResponseWriter, req *http.Request) (string, error) {
	totID := req.PathValue("id")
	if !isValidID(totID) {
		return totID, errors.New("invalid tot id")
	}

	flashKey := ""
	if cookie, err := req.Cookie("flash_msg"); err == nil {
		flashKey = cookie.Value
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	}

	data, err := s.getTotPageData(totID, flashKey)
	if err != nil {
		return totID, err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return totID, s.templateTot.Execute(w, data)
}

// manifestHandler returns a dynamic Web App Manifest.
// If an 'id' query parameter is present, it sets start_url to that tot's page.
// This ensures that 'Add to Home Screen' on iOS correctly points to the specific tot.
func (s *Server) manifestHandler(w http.ResponseWriter, req *http.Request) (string, error) {
	totID := req.URL.Query().Get("id")
	startURL := "/"
	if totID != "" && isValidID(totID) {
		startURL = "/" + totID
	}

	w.Header().Set("Content-Type", "application/manifest+json")
	fmt.Fprintf(w, `{
  "name": "Tot-Tally",
  "short_name": "Tot-Tally",
  "description": "A simple tracker for your tot's daily life.",
  "start_url": "%s",
  "display": "standalone",
  "icons": [
    {
      "src": "/favicon.ico",
      "sizes": "64x64",
      "type": "image/x-icon"
    }
  ]
}
`, startURL)
	return "", nil
}

func (s *Server) createTotHandler(w http.ResponseWriter, req *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		ip = req.RemoteAddr
	}

	if err := s.store.CheckAndIncrementIPLimit(ip); err != nil {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "error_limit_ip", Path: "/", MaxAge: 30, HttpOnly: true})
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return "", nil
	}

	name := req.FormValue("name")
	tz := req.FormValue("timezone")
	ms := req.FormValue("milk_setting")
	if ms == "" {
		ms = "both"
	}

	if _, okA := totConfig.AllowedAvatars[name]; !okA {
		return "", errors.New("invalid avatar")
	}
	if _, okT := totConfig.AllowedTimezones[tz]; !okT {
		return "", errors.New("invalid timezone")
	}
	if _, okM := totConfig.AllowedMilkSettings[ms]; !okM {
		return "", errors.New("invalid milk setting")
	}

	newID, err := s.core.CreateTot(name, tz, ms)
	if err != nil {
		return "", err
	}

	http.Redirect(w, req, "/"+newID, http.StatusSeeOther)
	return newID, nil
}

func (s *Server) updateTotHandler(w http.ResponseWriter, req *http.Request) (string, error) {
	totID := req.PathValue("id")
	mut := s.shards.GetShardMutex(totID)
	mut.Lock()
	defer mut.Unlock()

	tot, err := s.store.LoadTot(totID)
	if err != nil {
		return totID, err
	}

	tzLoc, _ := time.LoadLocation(tot.Timezone)
	changed, flashKey := false, ""

	if val := req.FormValue("tally"); val != "" {
		if err := s.core.AddTally(tot, val); err == nil {
			changed, flashKey = true, "tally"
		}
	} else if req.FormValue("delete_tot") != "" {
		if req.FormValue("confirm_delete") == "true" {
			filename := filepath.Join(s.config.TotDirectory, filepath.Base(totID)+".json")
			if err := os.Remove(filename); err != nil {
				return totID, fmt.Errorf("web: failed to delete tot file: %w", err)
			}
			http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: "deleted", Path: "/", MaxAge: 30, HttpOnly: true})
			http.Redirect(w, req, "/", http.StatusSeeOther)
			return "", nil
		}
	} else if req.FormValue("undo") != "" {
		if len(tot.Tallies) > 0 {
			tot.Tallies = tot.Tallies[1:]
			s.stats.RecalculateStats(tot)
			changed, flashKey = true, "undo"
		}
	} else if tz := req.FormValue("timezone"); tz != "" {
		if _, ok := totConfig.AllowedTimezones[tz]; ok {
			tot.Timezone = tz
			tzLoc, _ = time.LoadLocation(tot.Timezone)
			changed, flashKey = true, "updated"
		}
	} else if ms := req.FormValue("milk_setting"); ms != "" {
		if _, ok := totConfig.AllowedMilkSettings[ms]; ok {
			tot.MilkSetting = ms
			changed, flashKey = true, "updated"
		}
	}

	if changed {
		generated, err := s.stats.GenerateStats(tot, tzLoc, time.Now())
		if err != nil {
			return totID, err
		}
		tot.GeneratedStats = generated
		if err := s.store.SaveTot(tot); err != nil {
			return totID, err
		}
	}

	if flashKey != "" {
		http.SetCookie(w, &http.Cookie{Name: "flash_msg", Value: flashKey, Path: "/", MaxAge: 30, HttpOnly: true})
	}
	http.Redirect(w, req, "/"+totID, http.StatusSeeOther)
	return totID, nil
}

func (s *Server) exportTotHandler(w http.ResponseWriter, req *http.Request) (string, error) {
	totID := req.PathValue("id")
	filename := filepath.Join(s.config.TotDirectory, filepath.Base(totID)+".json")
	file, err := os.Open(filename)
	if err != nil {
		return totID, errors.New("tot does not exist")
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"tot-backup-%s.json\"", totID[:8]))
	w.Header().Set("Content-Type", "application/json")
	http.ServeContent(w, req, filename, time.Now(), file)
	return totID, nil
}

func (s *Server) getTotPageData(totID, flashKey string) (totModels.TotPageData, error) {
	tot, err := s.store.LoadTot(totID)
	if err != nil {
		return totModels.TotPageData{}, err
	}

	tz, _ := time.LoadLocation(tot.Timezone)
	formatted := make([]totModels.TotPageTally, len(tot.Tallies))
	for i := range tot.Tallies {
		t := &tot.Tallies[i]
		formatted[i] = totModels.TotPageTally{Time: t.Time.In(tz).Format(s.config.TimeFormat), Kind: t.Kind}
	}

	lastAmt := ""
	for i := range tot.Tallies {
		if amt, ok := strings.CutPrefix(tot.Tallies[i].Kind, "🍼"); ok {
			lastAmt = amt
			break
		}
	}

	displayMilk := tot.MilkSetting
	if len(displayMilk) > 0 {
		displayMilk = strings.ToUpper(displayMilk[:1]) + displayMilk[1:]
	}

	return totModels.TotPageData{
		ID: tot.ID, Name: tot.Name, Timezone: tot.Timezone, MilkSetting: tot.MilkSetting,
		MilkSettingDisplay: displayMilk, FlashMessage: totConfig.FlashMessages[flashKey],
		IsErrorFlash: strings.HasPrefix(totConfig.FlashMessages[flashKey], "Error:"),
		Tallies:      formatted, GeneratedStats: tot.GeneratedStats, MaxTallies: s.config.MaxTallies,
		Stats: totModels.TotPageStats{
			LastMilk: formatRelativeTime(tot.Stats.LastMilk), LastMilkAmount: lastAmt,
			LastNurse: formatRelativeTime(tot.Stats.LastNurse), LastNurseSide: tot.Stats.LastNurseSide,
			LastSnack: formatRelativeTime(tot.Stats.LastSnack), LastMeal: formatRelativeTime(tot.Stats.LastMeal),
			LastPee: formatRelativeTime(tot.Stats.LastPee), LastPoo: formatRelativeTime(tot.Stats.LastPoo),
			LastBath: formatRelativeTime(tot.Stats.LastBath), LastBrush: formatRelativeTime(tot.Stats.LastBrush),
		},
	}, nil
}

func formatRelativeTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "not yet"
	}
	d := time.Since(*t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh ago", h)
		}
		return fmt.Sprintf("%dh %dm ago", h, m)
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}
