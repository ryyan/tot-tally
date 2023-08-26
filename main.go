package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tot-tally/totdb"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

const (
	// Port to run the server on
	Port = ":5000"

	// Unique ID/Primary Key size
	IdSize = 33
)

var (
	// TODO: Use these before deploying to prod
	templateIndex = template.Must(template.ParseFiles("assets/index.html"))
	templateTally = template.Must(template.ParseFiles("assets/tally.html"))

	// https://docs.sqlc.dev/en/latest/tutorials/getting-started-sqlite.html
	//go:embed schema.sql
	ddl string

	// Needed for database and queries
	ctx = context.Background()

	// To be assigned at server initialization
	queries *totdb.Queries
)

type TallyPageData struct {
	Name              string
	Timezone          string
	Feeds             []Feed
	Soils             []Soil
	TimeSinceLastFeed string
	TimeSinceLastSoil string
}

type Feed struct {
	Time   string
	Note   string
	Ounces string
}

type Soil struct {
	Time string
	Note string
	Wet  string
	Soil string
}

func main() {
	// Initalize database
	initializeDatabase()

	// Set endpoints
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", handlerWrapper(rootHandler))

	fileServer := http.FileServer(http.Dir("assets/static/"))
	serveMux.Handle("/favicon.ico", http.StripPrefix("", fileServer))
	serveMux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Start server
	log.Printf("Server started at http://localhost:%s\n", Port)
	log.Fatal(http.ListenAndServe(Port, serveMux))
}

func initializeDatabase() {
	db, err := sql.Open("sqlite3", "totdb.db")
	check(err)

	// Create database tables
	_, err = db.ExecContext(ctx, ddl)
	if err != nil && err.Error() != "table babies already exists" {
		log.Fatal(err)
	}

	// Enable queries
	queries = totdb.New(db)
}

// https://thingsthatkeepmeupatnight.dev/posts/golang-http-handler-errors/
type HandlerE = func(w http.ResponseWriter, r *http.Request) error

func handlerWrapper(handler HandlerE) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Printf("%s %s\n", req.Method, req.URL.Path)
		err := handler(w, req)

		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())
			http.Redirect(w, req, "/", http.StatusSeeOther)
		}
	}
}

func rootHandler(w http.ResponseWriter, req *http.Request) error {
	// Ex: http://localhost:5000/abc123 --> baby_id=abc123
	baby_id := strings.TrimLeft(req.URL.Path, "/")

	switch req.Method {
	case http.MethodGet:
		if baby_id == "" {
			// Index page
			templateIndex.Execute(w, nil)
		} else {
			// Tally page
			data, err := getTallyPageData(baby_id)
			if err != nil {
				return err
			}

			templateTally.Execute(w, data)
		}
	case http.MethodPost:
		// POST means we're creating a new item, so generate its ID
		new_id, err := generateId()
		if err != nil {
			return err
		}

		if baby_id == "" {
			// Create new baby
			name := req.FormValue("name")
			timezone := req.FormValue(("timezone"))

			if strings.TrimSpace(name) == "" {
				return errors.New("name cannot be empty")
			}

			log.Printf("CreateBaby id=%s, name=%s timezone=%s\n", new_id, name, timezone)
			newBaby, err := queries.CreateBaby(ctx, totdb.CreateBabyParams{ID: new_id, Name: name, Timezone: timezone})
			if err != nil {
				return err
			}

			baby_id = newBaby.ID
		} else {
			if req.FormValue("ounces") != "" {
				// Create new feed
				err := createFeed(req, new_id, baby_id)
				if err != nil {
					return err
				}
			} else if req.FormValue("soil") != "" {
				// Create new soil
				err := createSoil(req, new_id, baby_id)
				if err != nil {
					return err
				}
			} else if req.FormValue("timezone") != "" {
				// Update timezone
				err := updateTimezone(req, baby_id)
				if err != nil {
					return err
				}
			}
		}

		// After POST, redirect to http://<url>/<baby_id>
		new_url := "/" + baby_id
		http.Redirect(w, req, new_url, http.StatusSeeOther)
	}

	return nil
}

func getTallyPageData(baby_id string) (TallyPageData, error) {
	// Get baby
	log.Printf("GetBaby id=%s\n", baby_id)
	getBaby, err := queries.GetBaby(ctx, baby_id)
	if err != nil {
		return TallyPageData{}, err
	}

	tzLocation, err := time.LoadLocation(getBaby.Timezone)
	if err != nil {
		return TallyPageData{}, err
	}

	// Get and format list of Feeds
	log.Printf("ListFeeds baby_id=%s\n", baby_id)
	listFeeds, err := queries.ListFeeds(ctx, baby_id)
	if err != nil {
		return TallyPageData{}, err
	}

	formattedFeeds := make([]Feed, len(listFeeds))
	for i, feed := range listFeeds {
		formattedTime := feed.CreatedAt.In(tzLocation).Format("2006-01-02 03:04 PM")
		ounces_str := strconv.FormatInt(feed.Ounces, 10)
		formattedFeeds[i] = Feed{formattedTime, feed.Note, ounces_str}
	}

	timeSinceLastFeed := "N/A"
	if len(listFeeds) > 0 {
		timeSinceLastFeed = timeElapsed(time.Now().UTC(), listFeeds[0].CreatedAt, false)
	}

	// Get and format list of Soils
	log.Printf("ListSoils baby_id=%s\n", baby_id)
	listSoils, err := queries.ListSoils(ctx, baby_id)
	if err != nil {
		return TallyPageData{}, err
	}

	timeSinceLastSoil := "N/A"
	if len(listSoils) > 0 {
		timeSinceLastSoil = timeElapsed(time.Now().UTC(), listSoils[0].CreatedAt, false)
	}

	formattedSoils := make([]Soil, len(listSoils))
	for i, soil := range listSoils {
		formattedTime := soil.CreatedAt.In(tzLocation).Format("2006-01-02 03:04 PM")
		formattedSoils[i] = Soil{formattedTime, soil.Note, soil.Wet, soil.Soil}
	}

	data := TallyPageData{Name: getBaby.Name, Timezone: getBaby.Timezone, Feeds: formattedFeeds, Soils: formattedSoils, TimeSinceLastFeed: timeSinceLastFeed, TimeSinceLastSoil: timeSinceLastSoil}
	return data, nil
}

func createFeed(req *http.Request, new_id string, baby_id string) error {
	note := req.FormValue("note")
	ounces := req.FormValue("ounces")
	ounces_str, err := strconv.ParseInt(ounces, 10, 64)
	if err != nil {
		return err
	}

	log.Printf("CreateFeed id=%s, baby_id=%s, note=%s, ounces=%s", new_id, baby_id, note, ounces)
	_, err = queries.CreateFeed(ctx, totdb.CreateFeedParams{new_id, baby_id, time.Now().UTC(), note, ounces_str})
	if err != nil {
		return err
	}

	return nil
}

func createSoil(req *http.Request, new_id string, baby_id string) error {
	note := req.FormValue("note")
	wet := req.FormValue("wet")
	soil := req.FormValue("soil")

	log.Printf("CreateSoil id=%s, baby_id=%s, note=%s, wet=%s, soil=%s", new_id, baby_id, note, wet, soil)
	_, err := queries.CreateSoil(ctx, totdb.CreateSoilParams{ID: new_id, BabyID: baby_id, CreatedAt: time.Now().UTC(), Note: note, Wet: wet, Soil: soil})
	if err != nil {
		return err
	}

	return nil
}

func updateTimezone(req *http.Request, baby_id string) error {
	timezone := req.FormValue("timezone")

	log.Printf("UpdateTimezone id=%s, timezone=%s", baby_id, timezone)
	_, err := queries.UpdateTimezone(ctx, totdb.UpdateTimezoneParams{ID: baby_id, Timezone: timezone})
	if err != nil {
		return err
	}

	return nil
}

// returns a URL-safe, base64 encoded securely generated random string
func generateId() (string, error) {
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

func s(x float64) string {
	if int(x) == 1 {
		return ""
	}
	return "s"
}

// https://socketloop.com/tutorials/golang-human-readable-time-elapsed-format-such-as-5-days-ago
func timeElapsed(now time.Time, then time.Time, full bool) string {
	var parts []string
	var text string

	year2, month2, day2 := now.Date()
	hour2, minute2, second2 := now.Clock()

	year1, month1, day1 := then.Date()
	hour1, minute1, second1 := then.Clock()

	year := math.Abs(float64(int(year2 - year1)))
	month := math.Abs(float64(int(month2 - month1)))
	day := math.Abs(float64(int(day2 - day1)))
	hour := math.Abs(float64(int(hour2 - hour1)))
	minute := math.Abs(float64(int(minute2 - minute1)))
	second := math.Abs(float64(int(second2 - second1)))

	week := math.Floor(day / 7)

	if year > 0 {
		parts = append(parts, strconv.Itoa(int(year))+" year"+s(year))
	}

	if month > 0 {
		parts = append(parts, strconv.Itoa(int(month))+" month"+s(month))
	}

	if week > 0 {
		parts = append(parts, strconv.Itoa(int(week))+" week"+s(week))
	}

	if day > 0 {
		parts = append(parts, strconv.Itoa(int(day))+" day"+s(day))
	}

	if hour > 0 {
		parts = append(parts, strconv.Itoa(int(hour))+" hour"+s(hour))
	}

	if minute > 0 {
		parts = append(parts, strconv.Itoa(int(minute))+" minute"+s(minute))
	}

	if second > 0 {
		parts = append(parts, strconv.Itoa(int(second))+" second"+s(second))
	}

	if now.After(then) {
		text = " ago"
	} else {
		text = " after"
	}

	if len(parts) == 0 {
		return "just now"
	}

	if full {
		return strings.Join(parts, ", ") + text
	}
	return parts[0] + text
}
