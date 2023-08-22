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
)

const (
	// Port to run the server on
	Port = ":5000"

	// TallyDir is the directory that tallies will be saved to
	// The filenames will be <random ID>-<kind of tally, ex: feed).csv
	TallyDir = ".tally"

	FeedFileSuffix   = "feed"
	DiaperFileSuffix = "diaper"

	// Unique ID size
	IdSize = 33
)

var (
	// TODO: Use these before deploying to prod
	//templateIndex = template.Must(template.ParseFiles("assets/index.html"))
	//templateTally = template.Must(template.ParseFiles("assets/tally.html"))

	// https://docs.sqlc.dev/en/latest/tutorials/getting-started-sqlite.html
	//go:embed schema.sql
	ddl string

	// Needed for database and queries
	ctx = context.Background()

	// To be assigned at server initialization
	queries *totdb.Queries
)

type TallyPageData struct {
	Name  string
	Feeds []Feed
	Soils []Soil
}

type Feed struct {
	Time   string
	Ounces string
	Note   string
}

type Soil struct {
	Time string
	Wet  string
	Soil string
	Note string
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
	db, err := sql.Open("sqlite3", ":memory:")
	check(err)

	// Create database tables
	_, err = db.ExecContext(ctx, ddl)
	check(err)

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
			templateIndex := template.Must(template.ParseFiles("assets/index.html"))
			templateIndex.Execute(w, nil)
		} else {
			// Tally page
			log.Printf("GetBaby id=%s\n", baby_id)
			getBaby, err := queries.GetBaby(ctx, baby_id)
			if err != nil {
				return err
			}

			tzLocation, err := time.LoadLocation(getBaby.Timezone)
			if err != nil {
				return err
			}

			log.Printf("ListFeeds baby_id=%s\n", baby_id)
			listFeeds, err := queries.ListFeeds(ctx, baby_id)
			if err != nil {
				return err
			}

			formatted_feeds := make([]Feed, len(listFeeds))
			for i, feed := range listFeeds {
				formatted_time := feed.CreatedAt.In(tzLocation).Format("2006-01-02 03:04 PM")
				ounces_str := strconv.FormatInt(feed.Ounces, 10)
				formatted_feeds[i] = Feed{formatted_time, ounces_str, feed.Note}
			}

			data := TallyPageData{Name: getBaby.Name, Feeds: formatted_feeds}
			templateTally := template.Must(template.ParseFiles("assets/tally.html"))
			templateTally.Execute(w, data)
		}
	case http.MethodPost:
		if baby_id == "" {
			// Create new baby
			name := req.FormValue("name")
			timezone := req.FormValue(("timezone"))

			if strings.TrimSpace(name) == "" {
				return errors.New("name cannot be empty")
			}

			new_baby_id, err := generateId()
			if err != nil {
				return err
			}

			log.Printf("CreateBaby id=%s, name=%s timezone=%s\n", new_baby_id, name, timezone)
			newBaby, err := queries.CreateBaby(ctx, totdb.CreateBabyParams{ID: new_baby_id, Name: name, Timezone: timezone})
			if err != nil {
				return err
			}

			new_url := "/" + newBaby.ID
			http.Redirect(w, req, new_url, http.StatusSeeOther)
		} else {
			// Create new feed/soil tally
			note := req.FormValue("note")
			ounces := req.FormValue("ounces")
			ounces_str, err := strconv.ParseInt(ounces, 10, 64)
			if err != nil {
				panic(err)
			}

			new_id, err := generateId()
			if err != nil {
				return err
			}

			log.Printf("CreateFeed id=%s, baby_id=%s, note=%s, ounces=%s", new_id, baby_id, note, ounces)
			_, err = queries.CreateFeed(ctx, totdb.CreateFeedParams{new_id, baby_id, time.Now().UTC(), note, ounces_str})
			if err != nil {
				return err
			}

			new_url := "/" + baby_id
			http.Redirect(w, req, new_url, http.StatusSeeOther)
		}
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
