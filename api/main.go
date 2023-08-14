package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// Port to run the server on
	Port = ":5000"

	// TallyDir is the directory that tallies will be saved to
	// The filenames will be <random ID>-<kind of tally, ex: feed).csv
	TallyDir = ".tally"

	FeedFileSuffix   = "feed"
	DiaperFileSuffix = "diaper"
)

func main() {
	// Set endpoints
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/api/", handler)
	serveMux.HandleFunc("/api/feed", feedHandler)
	serveMux.HandleFunc("/api/diaper", diaperHandler)

	// Start server
	log.Printf("Server started at http://localhost:%s\n", Port)
	log.Fatal(http.ListenAndServe(Port, serveMux))
}

func handler(res http.ResponseWriter, req *http.Request) {
	var result string
	var err error

	switch req.Method {
	case "GET":
		id := strings.TrimLeft(req.URL.Path, "api/")
		result = id
	case "POST":
		id := generateId(42)
		err = newFeedFile(id)
		if err != nil {
			break
		}

		err = newDiaperFile(id)
		result = id
	}

	writeResponse(res, result, err)
}

func feedHandler(res http.ResponseWriter, req *http.Request) {
	var result string
	var err error

	switch req.Method {
	case "POST":
		req.ParseForm()
		id := req.FormValue("id")
		ounces := req.FormValue("ounces")
		err = newFeed(id, ounces)
	}

	writeResponse(res, result, err)
}

func diaperHandler(res http.ResponseWriter, req *http.Request) {
	var result string
	var err error

	switch req.Method {
	case "POST":
		req.ParseForm()
		id := req.FormValue("id")
		wet := req.FormValue("wet")
		soil := req.FormValue("soil")
		err = newDiaper(id, wet, soil)
	}

	writeResponse(res, result, err)
}

func generateTallyFilename(id string, suffix string) string {
	// Ex: abc123-feed.csv
	return id + "-" + suffix + ".csv"
}

func newFeedFile(id string) error {
	filename := generateTallyFilename(id, FeedFileSuffix)
	headerRow := []string{"epoch", "ounces"}
	return writeRowToCsv(filename, headerRow, true)
}

func newFeed(id, ounces string) error {
	now := time.Now()
	filename := generateTallyFilename(id, FeedFileSuffix)
	row := []string{fmt.Sprint(now.Unix()), ounces}
	return writeRowToCsv(filename, row, false)
}

func newDiaperFile(id string) error {
	filename := generateTallyFilename(id, DiaperFileSuffix)
	headerRow := []string{"epoch", "wet", "soil"}
	return writeRowToCsv(filename, headerRow, true)
}

func newDiaper(id, wet, soil string) error {
	now := time.Now()
	filename := generateTallyFilename(id, DiaperFileSuffix)
	row := []string{fmt.Sprint(now.Unix()), wet, soil}
	return writeRowToCsv(filename, row, false)
}

func writeRowToCsv(filename string, row []string, isNewFile bool) error {
	var file *os.File
	var err error

	if isNewFile {
		file, err = os.Create(filename)
	} else {
		file, err = os.Open(filename)
	}

	if err != nil {
		return err
	}

	defer file.Close()

	w := csv.NewWriter(file)
	err = w.Write(row)
	if err != nil {
		return err
	}

	// Write any buffered data to the underlying writer (standard output)
	w.Flush()

	err = w.Error()
	if err != nil {
		return err
	}

	return nil
}

func writeResponse(res http.ResponseWriter, result string, err error) {
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		io.WriteString(res, err.Error())
	} else {
		io.WriteString(res, result)
	}
}

// check reduces the amount of if err != nil spam
func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var (
	random = rand.NewSource(time.Now().UTC().UnixNano())
)

// generateId generates a random string
// https://stackoverflow.com/questions/22892120
func generateId(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i, cache, remain := n-1, random.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = random.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}
