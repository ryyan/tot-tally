package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tot-tally/totdb"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"
	elapsed "github.com/rvflash/elapsed"
)

const (
	// Port to run the server on
	Port = ":5000"

	// Unique ID/Primary Key size
	IdSize = 33
)

var (
	templateIndex = template.Must(template.ParseFiles("assets/index.html"))
	templateTally = template.Must(template.ParseFiles("assets/tally.html"))

	feedTypeMap = map[int64]string{
		0: "🍼",
		1: "🥫",
	}

	soilBoolMap = map[int64]string{
		0: "❌",
		1: "✔️",
	}

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
	TimeSinceLastMilk string
	TimeSinceLastFood string
	TimeSinceLastWet  string
	TimeSinceLastSoil string
}

type Feed struct {
	Time     string
	Ounces   string
	FeedType string
}

type Soil struct {
	Time string
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
type HandlerE = func(w http.ResponseWriter, r *http.Request) (string, error)

func handlerWrapper(handler HandlerE) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Printf("%s %s\n", req.Method, req.URL.Path)
		babyID, err := handler(w, req)

		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())

			if babyID == "" || err.Error() == "baby does not exist" {
				http.Redirect(w, req, "/", http.StatusSeeOther)
			} else {
				http.Redirect(w, req, "/"+babyID, http.StatusSeeOther)
			}
		}
	}
}

func rootHandler(w http.ResponseWriter, req *http.Request) (string, error) {
	// Ex: http://localhost:5000/abc123 --> babyID=abc123
	babyID := strings.TrimLeft(req.URL.Path, "/")

	switch req.Method {
	case http.MethodGet:
		if babyID == "" {
			// Index page
			templateIndex.Execute(w, nil)
		} else {
			// Tally page
			data, err := getTallyPageData(babyID)
			if err != nil {
				return babyID, err
			}

			templateTally.Execute(w, data)
		}
	case http.MethodPost:
		if babyID == "" {
			// Create new baby
			newBabyId, err := createBaby(req)
			if err != nil {
				return babyID, err
			}
			babyID = newBabyId
		} else {
			if req.FormValue("ounces") != "" {
				// Create new feed
				err := createFeed(req, babyID)
				if err != nil {
					return babyID, err
				}
			} else if req.FormValue("soil") != "" {
				// Create new soil
				err := createSoil(req, babyID)
				if err != nil {
					return babyID, err
				}
			} else if req.FormValue("timezone") != "" {
				// Update timezone
				err := updateTimezone(req, babyID)
				if err != nil {
					return babyID, err
				}
			}
		}

		// After POST, redirect to http://<url>/<babyID>
		redirectURL := "/" + babyID
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
	}

	return babyID, nil
}

func getTallyPageData(babyID string) (TallyPageData, error) {
	// Get baby
	log.Printf("GetBaby id=%s\n", babyID)
	getBaby, err := queries.GetBaby(ctx, babyID)
	if err != nil {
		return TallyPageData{}, errors.New("baby does not exist")
	}

	tzLocation, err := time.LoadLocation(getBaby.Timezone)
	if err != nil {
		return TallyPageData{}, err
	}

	// Get and format list of Feeds
	log.Printf("ListFeeds babyID=%s\n", babyID)
	listFeeds, err := queries.ListFeeds(ctx, babyID)
	if err != nil {
		return TallyPageData{}, err
	}

	formattedFeeds := make([]Feed, len(listFeeds))
	for i, feed := range listFeeds {
		formattedTime := feed.CreatedAt.In(tzLocation).Format("2006-01-02 03:04 PM")
		ouncesString := strconv.FormatInt(feed.Ounces, 10)
		typeString := feedTypeMap[feed.FeedType]
		formattedFeeds[i] = Feed{Time: formattedTime, FeedType: typeString, Ounces: ouncesString}
	}

	// Get and format list of Soils
	log.Printf("ListSoils babyID=%s\n", babyID)
	listSoils, err := queries.ListSoils(ctx, babyID)
	if err != nil {
		return TallyPageData{}, err
	}

	formattedSoils := make([]Soil, len(listSoils))
	for i, soil := range listSoils {
		formattedTime := soil.CreatedAt.In(tzLocation).Format("2006-01-02 03:04 PM")
		formattedSoils[i] = Soil{Time: formattedTime, Wet: soilBoolMap[soil.Wet], Soil: soilBoolMap[soil.Soil]}
	}

	// Get and generate human-readable "time since last X"
	lastMilkTime, err := queries.GetLastMilkTime(ctx, babyID)
	timeSinceLastMilk := "not yet"
	if err == nil {
		timeSinceLastMilk = elapsed.Time(lastMilkTime.In(tzLocation))
	}

	lastFoodTime, err := queries.GetLastFoodTime(ctx, babyID)
	timeSinceLastFood := "not yet"
	if err == nil {
		timeSinceLastFood = elapsed.Time(lastFoodTime.In(tzLocation))
	}

	lastWetTime, err := queries.GetLastWetTime(ctx, babyID)
	timeSinceLastWet := "not yet"
	if err == nil {
		timeSinceLastWet = elapsed.Time(lastWetTime.In(tzLocation))
	}

	lastSoilTime, err := queries.GetLastSoilTime(ctx, babyID)
	timeSinceLastSoil := "not yet"
	if err == nil {
		timeSinceLastSoil = elapsed.Time(lastSoilTime.In(tzLocation))
	}

	data := TallyPageData{Name: getBaby.Name, Timezone: getBaby.Timezone, Feeds: formattedFeeds, Soils: formattedSoils, TimeSinceLastMilk: timeSinceLastMilk, TimeSinceLastFood: timeSinceLastFood, TimeSinceLastWet: timeSinceLastWet, TimeSinceLastSoil: timeSinceLastSoil}
	return data, nil
}

func createBaby(req *http.Request) (string, error) {
	name := req.FormValue("name")
	timezone := req.FormValue(("timezone"))
	log.Printf("CreateBaby name=%s timezone=%s\n", name, timezone)

	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("name cannot be empty")
	}
	if len(name) > 20 {
		return "", errors.New("name must be less than 20 characters")
	}

	newID, err := generateID()
	if err != nil {
		return "", err
	}

	log.Printf("CreateBaby newID=%s\n", newID)
	newBaby, err := queries.CreateBaby(ctx, totdb.CreateBabyParams{ID: newID, Name: name, Timezone: timezone})
	if err != nil {
		return "", err
	}

	return newBaby.ID, nil
}

func createFeed(req *http.Request, babyID string) error {
	feedType := req.FormValue("feedType")
	ounces := req.FormValue("ounces")
	log.Printf("CreateFeed babyID=%s, feedType=%s, ounces=%s", babyID, feedType, ounces)

	feedTypeInt, err := strconv.ParseInt(feedType, 10, 64)
	if err != nil {
		return err
	}

	if _, exists := feedTypeMap[feedTypeInt]; exists == false {
		return errors.New("invalid feed type")
	}

	ouncesInt, err := strconv.ParseInt(ounces, 10, 64)
	if err != nil {
		return err
	}

	if ouncesInt < 1 || ouncesInt > 9 {
		return errors.New("ounces must be between 1 and 9")
	}

	_, err = queries.CreateFeed(ctx, totdb.CreateFeedParams{BabyID: babyID, CreatedAt: time.Now().UTC(), FeedType: feedTypeInt, Ounces: ouncesInt})
	if err != nil {
		return err
	}

	return nil
}

func createSoil(req *http.Request, babyID string) error {
	wet := req.FormValue("wet")
	soil := req.FormValue("soil")
	log.Printf("CreateSoil babyID=%s, wet=%s, soil=%s", babyID, wet, soil)

	wetInt, err := strconv.ParseInt(wet, 10, 64)
	if err != nil {
		return err
	}

	if wetInt != 0 && wetInt != 1 {
		return errors.New("wet must be 0 or 1")
	}

	soilInt, err := strconv.ParseInt(soil, 10, 64)
	if err != nil {
		return err
	}

	if soilInt != 0 && soilInt != 1 {
		return errors.New("soil must be 0 or 1")
	}

	if wetInt == 0 && soilInt == 0 {
		return errors.New("must have either wet, soil, or both")
	}

	_, err = queries.CreateSoil(ctx, totdb.CreateSoilParams{BabyID: babyID, CreatedAt: time.Now().UTC(), Wet: wetInt, Soil: soilInt})
	if err != nil {
		return err
	}

	return nil
}

func updateTimezone(req *http.Request, babyID string) error {
	timezone := req.FormValue("timezone")
	log.Printf("UpdateTimezone id=%s, timezone=%s", babyID, timezone)

	_, err := queries.UpdateTimezone(ctx, totdb.UpdateTimezoneParams{ID: babyID, Timezone: timezone})
	if err != nil {
		return err
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
