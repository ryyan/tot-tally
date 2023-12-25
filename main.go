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

	totdb "tot-tally/generated-sql"

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

	tallyKindMap = map[int64]string{
		1:  "Milk 1oz",
		2:  "Milk 2oz",
		3:  "Milk 3oz",
		4:  "Milk 4oz",
		5:  "Milk 5oz",
		6:  "Milk 6oz",
		7:  "Milk 7oz",
		8:  "Milk 8oz",
		9:  "Food (Snack)",
		10: "Food (Meal)",
		11: "Wet",
		12: "Soil",
		13: "Wet & Soil",
		14: "Bath",
		15: "Toothbrush",
	}

	// https://docs.sqlc.dev/en/latest/tutorials/getting-started-sqlite.html
	//go:embed schema.sql
	ddl string

	// Needed for database and queries
	ctx = context.Background()

	// To be assigned at database initialization
	queries *totdb.Queries
)

type TallyPageData struct {
	Name                    string
	Timezone                string
	Tallies                 []Tally
	TimeSinceLastMilk       string
	TimeSinceLastSnack      string
	TimeSinceLastMeal       string
	TimeSinceLastWet        string
	TimeSinceLastSoil       string
	TimeSinceLastBath       string
	TimeSinceLastToothbrush string
}

type Tally struct {
	Time string
	Kind string
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
	if err != nil && err.Error() != "table tots already exists" {
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

func rootHandler(w http.ResponseWriter, req *http.Request) (string, error) {
	// Ex: http://localhost:5000/abc123 --> totID=abc123
	totID := strings.TrimLeft(req.URL.Path, "/")

	switch req.Method {
	case http.MethodGet:
		if totID == "" {
			// Index page
			templateIndex.Execute(w, nil)
		} else {
			// Tally page
			data, err := getTallyPageData(totID)
			if err != nil {
				return totID, err
			}

			templateTally.Execute(w, data)
		}
	case http.MethodPost:
		if totID == "" {
			// Create new tot
			newTotId, err := createTot(req)
			if err != nil {
				return totID, err
			}
			totID = newTotId
		} else {
			if req.FormValue("tally") != "" {
				// Create new tally
				err := createTally(req, totID)
				if err != nil {
					return totID, err
				}
			} else if req.FormValue("timezone") != "" {
				// Update timezone
				err := updateTimezone(req, totID)
				if err != nil {
					return totID, err
				}
			}
		}

		// After POST, redirect to http://<url>/<totID>
		redirectURL := "/" + totID
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
	}

	return totID, nil
}

func getTallyPageData(totID string) (TallyPageData, error) {
	// Get tot
	log.Printf("GetTot id=%s\n", totID)
	tot, err := queries.GetTot(ctx, totID)
	if err != nil {
		return TallyPageData{}, errors.New("tot does not exist")
	}

	tzLocation, err := time.LoadLocation(tot.Timezone)
	if err != nil {
		return TallyPageData{}, err
	}

	// Get and format list of Feeds
	log.Printf("ListTallies totID=%s\n", totID)
	listTallies, err := queries.ListTallies(ctx, totID)
	if err != nil {
		return TallyPageData{}, err
	}

	formattedTallies := make([]Tally, len(listTallies))
	for i, feed := range listTallies {
		formattedTime := feed.CreatedAt.In(tzLocation).Format("Mon, Jan 02, 03:04 PM")
		formattedTallies[i] = Tally{Time: formattedTime, Kind: feed.Kind}
	}

	// Get and generate human-readable "time since last X"
	lastMilkTime, err := queries.GetLastMilkTime(ctx, totID)
	timeSinceLastMilk := "not yet"
	if err == nil {
		timeSinceLastMilk = elapsed.Time(lastMilkTime.In(tzLocation))
	}

	lastSnackTime, err := queries.GetLastSnackTime(ctx, totID)
	timeSinceLastSnack := "not yet"
	if err == nil {
		timeSinceLastSnack = elapsed.Time(lastSnackTime.In(tzLocation))
	}

	lastMealTime, err := queries.GetLastMealTime(ctx, totID)
	timeSinceLastMeal := "not yet"
	if err == nil {
		timeSinceLastMeal = elapsed.Time(lastMealTime.In(tzLocation))
	}

	lastWetTime, err := queries.GetLastWetTime(ctx, totID)
	timeSinceLastWet := "not yet"
	if err == nil {
		timeSinceLastWet = elapsed.Time(lastWetTime.In(tzLocation))
	}

	lastSoilTime, err := queries.GetLastSoilTime(ctx, totID)
	timeSinceLastSoil := "not yet"
	if err == nil {
		timeSinceLastSoil = elapsed.Time(lastSoilTime.In(tzLocation))
	}

	lastBathTime, err := queries.GetLastBathTime(ctx, totID)
	timeSinceLastBath := "not yet"
	if err == nil {
		timeSinceLastBath = elapsed.Time(lastBathTime.In(tzLocation))
	}

	lastToothbrushTime, err := queries.GetLastToothbrushTime(ctx, totID)
	timeSinceLastToothbrush := "not yet"
	if err == nil {
		timeSinceLastToothbrush = elapsed.Time(lastToothbrushTime.In(tzLocation))
	}

	data := TallyPageData{
		Name:                    tot.Name,
		Timezone:                tot.Timezone,
		Tallies:                 formattedTallies,
		TimeSinceLastMilk:       timeSinceLastMilk,
		TimeSinceLastSnack:      timeSinceLastSnack,
		TimeSinceLastMeal:       timeSinceLastMeal,
		TimeSinceLastWet:        timeSinceLastWet,
		TimeSinceLastSoil:       timeSinceLastSoil,
		TimeSinceLastBath:       timeSinceLastBath,
		TimeSinceLastToothbrush: timeSinceLastToothbrush,
	}
	log.Println(formattedTallies)
	log.Println(data)
	return data, nil
}

func createTot(req *http.Request) (string, error) {
	name := req.FormValue("name")
	timezone := req.FormValue(("timezone"))
	log.Printf("CreateTot name=%s timezone=%s\n", name, timezone)

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

	log.Printf("CreateTot newID=%s\n", newID)
	newTot, err := queries.CreateTot(ctx, totdb.CreateTotParams{ID: newID, Name: name, Timezone: timezone})
	if err != nil {
		return "", err
	}

	return newTot.ID, nil
}

func createTally(req *http.Request, totID string) error {
	kindKey := req.FormValue("tally")
	log.Printf("createTally totID=%s, kind=%s", totID, kindKey)

	kindKeyInt, err := strconv.ParseInt(kindKey, 10, 64)
	if err != nil {
		return err
	}

	kind, exists := tallyKindMap[kindKeyInt]
	if !exists {
		return errors.New("invalid tally kind")
	}

	_, err = queries.CreateTally(ctx, totdb.CreateTallyParams{TotID: totID, CreatedAt: time.Now().UTC(), Kind: kind})
	if err != nil {
		return err
	}

	return nil
}

func updateTimezone(req *http.Request, totID string) error {
	timezone := req.FormValue("timezone")
	log.Printf("UpdateTimezone id=%s, timezone=%s", totID, timezone)

	_, err := time.LoadLocation(timezone)
	if err != nil {
		return errors.New("invalid timezone")
	}

	_, err = queries.UpdateTimezone(ctx, totdb.UpdateTimezoneParams{ID: totID, Timezone: timezone})
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
