package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	// Port to run the server on
	Port = ":5000"

	// Unique ID/Primary Key size
	IdSize = 33

	// Directory to save Tot/Tallies to
	TotDirectory = "tots"

	// Maximum number of Tallies to save per Tot
	MaxTallies = 300

	TimeFormat = "Jan 02, 03:04PM"
)

var (
	templateIndex = template.Must(template.ParseFiles("assets/index.html"))
	templateTally = template.Must(template.ParseFiles("assets/tally.html"))

	tallyKindMap = map[int64]string{
		1:  "🍼1", // Milk (oz)
		2:  "🍼2",
		3:  "🍼3",
		4:  "🍼4",
		5:  "🍼5",
		6:  "🍼6",
		7:  "🍼7",
		8:  "🍼8",
		9:  "🍎",  // Snack
		10: "🍲",  // Meal
		11: "🚽",  // Pee
		12: "💩",  // Poo
		13: "🚽💩", // Pee & Poo
		14: "🛁",  // Bath
		15: "🦷",  // Toothbrush
	}
)

type Tot struct {
	ID       string
	Name     string
	Timezone string
	Tallies  []Tally
	Stats    Stats
}

type Tally struct {
	Time *time.Time
	Kind string
}

type Stats struct {
	LastMilk  *time.Time
	LastSnack *time.Time
	LastMeal  *time.Time
	LastPee   *time.Time
	LastPoo   *time.Time
	LastBath  *time.Time
	LastBrush *time.Time
}

type TotPageData struct {
	Name     string
	Timezone string
	Tallies  []TotPageTally
	Stats    TotPageStats
}

type TotPageTally struct {
	Time string
	Kind string
}

type TotPageStats struct {
	LastMilk  string
	LastSnack string
	LastMeal  string
	LastPee   string
	LastPoo   string
	LastBath  string
	LastBrush string
}

func main() {
	// Create tot file directory
	_ = os.Mkdir(TotDirectory, 0755)

	// Set endpoints
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("GET /", handlerWrapper(homeHandler))
	serveMux.HandleFunc("GET /{id}", handlerWrapper(getTotHandler))
	serveMux.HandleFunc("POST /", handlerWrapper(createTotHandler))
	serveMux.HandleFunc("POST /{id}", handlerWrapper(updateTotHandler))

	fileServer := http.FileServer(http.Dir("assets/static/"))
	serveMux.Handle("GET /favicon.ico", http.StripPrefix("", fileServer))
	//serveMux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Start server
	log.Printf("Server started at http://localhost:%s\n", Port)
	log.Fatal(http.ListenAndServe(Port, serveMux))
}

// https://thingsthatkeepmeupatnight.dev/posts/golang-http-handler-errors/
type HandlerE = func(w http.ResponseWriter, r *http.Request) (string, error)

func handlerWrapper(handler HandlerE) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Printf("%s %s\n", req.Method, req.URL.Path)
		totID, err := handler(w, req)

		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())

			if totID == "" || err.Error() == "tot does not exist" {
				http.Redirect(w, req, "/", http.StatusSeeOther)
			} else {
				http.Redirect(w, req, "/"+totID, http.StatusSeeOther)
			}
		}
	}
}

func homeHandler(w http.ResponseWriter, req *http.Request) (string, error) {
	templateIndex.Execute(w, nil)
	return "", nil
}

func getTotHandler(w http.ResponseWriter, req *http.Request) (string, error) {
	totID := req.PathValue("id")
	data, err := getTotPageData(totID)
	if err != nil {
		return totID, err
	}

	templateTally.Execute(w, data)
	return totID, nil
}

func createTotHandler(w http.ResponseWriter, req *http.Request) (string, error) {
	// Create new tot
	name := req.FormValue("name")
	timezone := req.FormValue(("timezone"))
	newTotId, err := createTot(name, timezone)
	if err != nil {
		return newTotId, err
	}

	// After POST, redirect to http://<url>/<totID>
	redirectURL := "/" + newTotId
	http.Redirect(w, req, redirectURL, http.StatusSeeOther)

	return newTotId, nil
}

func updateTotHandler(w http.ResponseWriter, req *http.Request) (string, error) {
	totID := req.PathValue("id")
	tot, err := loadTot(totID)
	if err != nil {
		return totID, err
	}

	if req.FormValue("tally") != "" {
		// Create new tally
		kindKey := req.FormValue("tally")
		err = createTally(tot, kindKey)
		if err != nil {
			return totID, err
		}
	} else if req.FormValue("timezone") != "" {
		// Update timezone
		timezone := req.FormValue("timezone")
		err = updateTimezone(tot, timezone)
		if err != nil {
			return totID, err
		}
	}

	err = saveTot(tot)
	if err != nil {
		return totID, err
	}

	// After POST, redirect to http://<url>/<totID>
	redirectURL := "/" + totID
	http.Redirect(w, req, redirectURL, http.StatusSeeOther)

	return totID, nil
}

func getTotPageData(totID string) (TotPageData, error) {
	// Get tot
	log.Printf("GetTot id=%s\n", totID)
	tot, err := loadTot(totID)
	if err != nil {
		log.Printf("ERROR: %s", err)
		return TotPageData{}, errors.New("tot does not exist")
	}

	tzLocation, err := time.LoadLocation(tot.Timezone)
	if err != nil {
		return TotPageData{}, err
	}

	// Format Tallies
	formattedTallies := make([]TotPageTally, len(tot.Tallies))
	for i, tally := range tot.Tallies {
		formattedTime := tally.Time.In(tzLocation).Format(TimeFormat)
		formattedTallies[i] = TotPageTally{Time: formattedTime, Kind: tally.Kind}
	}

	// Format stats
	formattedStats := TotPageStats{
		LastMilk:  formatTime(tot.Stats.LastMilk, tzLocation),
		LastSnack: formatTime(tot.Stats.LastSnack, tzLocation),
		LastMeal:  formatTime(tot.Stats.LastMeal, tzLocation),
		LastPee:   formatTime(tot.Stats.LastPee, tzLocation),
		LastPoo:   formatTime(tot.Stats.LastPoo, tzLocation),
		LastBath:  formatTime(tot.Stats.LastBath, tzLocation),
		LastBrush: formatTime(tot.Stats.LastBrush, tzLocation),
	}

	data := TotPageData{
		Name:     tot.Name,
		Timezone: tot.Timezone,
		Tallies:  formattedTallies,
		Stats:    formattedStats,
	}

	return data, nil
}

func formatTime(t *time.Time, tzLocation *time.Location) string {
	if t == nil || t.IsZero() {
		return "not yet"
	}

	return t.In(tzLocation).Format(TimeFormat)
}

func createTot(name string, timezone string) (string, error) {
	log.Printf("CreateTot name=%s timezone=%s\n", name, timezone)

	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("name cannot be empty")
	}
	if IsLetter(name) == false {
		return "", errors.New("name must be alphabetic characters only")
	}
	if len(name) > 15 {
		return "", errors.New("name must be less than 15 characters")
	}

	_, err := time.LoadLocation(timezone)
	if err != nil {
		return "", errors.New("invalid timezone")
	}

	newID, err := generateID()
	if err != nil {
		return "", err
	}
	log.Printf("CreateTot newID=%s\n", newID)

	newTot := Tot{
		ID:       newID,
		Name:     name,
		Timezone: timezone,
		Tallies:  []Tally{},
	}

	err = saveTot(&newTot)
	if err != nil {
		return "", err
	}

	return newID, nil
}

func createTally(tot *Tot, kindKey string) error {
	log.Printf("createTally totID=%s, kindKey=%s", tot.ID, kindKey)
	kind, err := parseKindKey(kindKey)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	prependTally(tot, kind, &now)
	updateStat(tot, kind, &now)
	return nil
}

func parseKindKey(kindKey string) (string, error) {
	kindKeyInt, err := strconv.ParseInt(kindKey, 10, 64)
	if err != nil {
		return "", err
	}

	kind, exists := tallyKindMap[kindKeyInt]
	if !exists {
		return "", errors.New("invalid tally kind")
	}

	return kind, nil
}

func prependTally(tot *Tot, kind string, now *time.Time) {
	newTallies := []Tally{}

	switch kind {
	case tallyKindMap[13]:
		// Record pee and poo seperately
		newTallies = append(newTallies, Tally{
			Time: now,
			Kind: tallyKindMap[11],
		})

		newTallies = append(newTallies, Tally{
			Time: now,
			Kind: tallyKindMap[12],
		})
	default:
		newTallies = append(newTallies, Tally{
			Time: now,
			Kind: kind,
		})
	}

	// Prepend to ensure latest are first
	tot.Tallies = append(newTallies, tot.Tallies...)

	// Enforce tally list count limit
	if len(tot.Tallies) > MaxTallies {
		tot.Tallies = tot.Tallies[:MaxTallies]
	}
}

func updateStat(tot *Tot, kind string, now *time.Time) {
	if strings.HasPrefix(kind, "🍼") {
		tot.Stats.LastMilk = now
		return
	}

	switch kind {
	case tallyKindMap[9]:
		tot.Stats.LastSnack = now
	case tallyKindMap[10]:
		tot.Stats.LastMeal = now
	case tallyKindMap[11]:
		tot.Stats.LastPee = now
	case tallyKindMap[12]:
		tot.Stats.LastPoo = now
	case tallyKindMap[13]:
		tot.Stats.LastPee = now
		tot.Stats.LastPoo = now
	case tallyKindMap[14]:
		tot.Stats.LastBath = now
	case tallyKindMap[15]:
		tot.Stats.LastBrush = now
	}
}

func updateTimezone(tot *Tot, timezone string) error {
	log.Printf("UpdateTimezone id=%s, timezone=%s", tot.ID, timezone)

	_, err := time.LoadLocation(timezone)
	if err != nil {
		return errors.New("invalid timezone")
	}

	return nil
}

// returns a URL-safe, base64 encoded securely generated random string
func generateID() (string, error) {
	b := make([]byte, IdSize)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

// check reduces the amount of if err != nil spam
func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func IsLetter(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func saveTot(tot *Tot) error {
	totJson, err := json.Marshal(tot)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s/%s.json", TotDirectory, tot.ID)
	return os.WriteFile(filename, totJson, 0755)
}

func loadTot(totID string) (*Tot, error) {
	filename := fmt.Sprintf("./%s/%s.json", TotDirectory, totID)
	log.Println(filename)
	totFile, err := os.ReadFile(filename)
	if err != nil {
		return &Tot{}, errors.New("Tot not found")
	}

	var tot *Tot = &Tot{}
	err = json.Unmarshal(totFile, tot)
	if err != nil {
		return &Tot{}, err
	}

	return tot, nil
}
