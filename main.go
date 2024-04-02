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

	"github.com/rvflash/elapsed"
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
)

var (
	templateIndex = template.Must(template.ParseFiles("assets/index.html"))
	templateTally = template.Must(template.ParseFiles("assets/tally.html"))

	tallyKindMap = map[int64]string{
		1:  "🍼 1", // Milk (oz)
		2:  "🍼 2",
		3:  "🍼 3",
		4:  "🍼 4",
		5:  "🍼 5",
		6:  "🍼 6",
		7:  "🍼 7",
		8:  "🍼 8",
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
}

type Tally struct {
	Time time.Time
	Kind string
}

type TotPageData struct {
	Name     string
	Timezone string
	Tallies  []TotPageTally

	TimeSinceLastMilk  string
	TimeSinceLastSnack string
	TimeSinceLastMeal  string
	TimeSinceLastPee   string
	TimeSinceLastPoo   string
	TimeSinceLastBath  string
	TimeSinceLastBrush string
}

type TotPageTally struct {
	Time string
	Kind string
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
	serveMux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

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
		formattedTime := tally.Time.In(tzLocation).Format("Jan 02, 03:04PM")
		formattedTallies[i] = TotPageTally{Time: formattedTime, Kind: tally.Kind}
	}

	// Get and generate human-readable "time since last X"
	timeSinceLastMilk := tot.lastTimePrefix(tzLocation, "🍼")
	timeSinceLastSnack := tot.lastTime(tzLocation, "🍎")
	timeSinceLastMeal := tot.lastTime(tzLocation, "🍲")
	timeSinceLastPee := tot.lastTime(tzLocation, "🚽")
	timeSinceLastPoo := tot.lastTime(tzLocation, "💩")
	timeSinceLastBath := tot.lastTime(tzLocation, "🛁")
	timeSinceLastBrush := tot.lastTime(tzLocation, "🦷")

	data := TotPageData{
		Name:               tot.Name,
		Timezone:           tot.Timezone,
		Tallies:            formattedTallies,
		TimeSinceLastMilk:  timeSinceLastMilk,
		TimeSinceLastSnack: timeSinceLastSnack,
		TimeSinceLastMeal:  timeSinceLastMeal,
		TimeSinceLastPee:   timeSinceLastPee,
		TimeSinceLastPoo:   timeSinceLastPoo,
		TimeSinceLastBath:  timeSinceLastBath,
		TimeSinceLastBrush: timeSinceLastBrush,
	}

	return data, nil
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

	kindKeyInt, err := strconv.ParseInt(kindKey, 10, 64)
	if err != nil {
		return err
	}

	kind, exists := tallyKindMap[kindKeyInt]
	if !exists {
		return errors.New("invalid tally kind")
	}

	newTallies := []Tally{}

	switch kindKeyInt {
	case 13:
		// Record pee and poo seperately
		newTallies = append(newTallies, Tally{
			Time: time.Now().UTC(),
			Kind: tallyKindMap[11],
		})

		newTallies = append(newTallies, Tally{
			Time: time.Now().UTC(),
			Kind: tallyKindMap[12],
		})
	default:
		newTallies = append(newTallies, Tally{
			Time: time.Now().UTC(),
			Kind: kind,
		})

	}

	// Prepend to ensure latest are first
	tot.Tallies = append(newTallies, tot.Tallies...)

	// Enforce tally list count limit
	if len(tot.Tallies) > MaxTallies {
		tot.Tallies = tot.Tallies[:MaxTallies]
	}

	return nil
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

func (tot *Tot) lastTimePrefix(tzLocation *time.Location, kind string) string {
	var lastTime time.Time

	for _, tally := range tot.Tallies {
		if strings.HasPrefix(tally.Kind, kind) {
			lastTime = tally.Time
		}
	}

	return elapsedTime(tzLocation, &lastTime)
}

func (tot *Tot) lastTime(tzLocation *time.Location, kind string) string {
	var lastTime time.Time

	for _, tally := range tot.Tallies {
		if tally.Kind == kind {
			lastTime = tally.Time
		}
	}

	return elapsedTime(tzLocation, &lastTime)
}

func elapsedTime(tzLocation *time.Location, time *time.Time) string {
	if time.IsZero() {
		return "not yet"
	} else {
		return elapsed.Time(time.In(tzLocation))
	}
}
