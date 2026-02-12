package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/data"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"healthy"}`)
	})

	// Match NOAA URL pattern: /{YYMMDD}_rpts_{type}.csv
	// Serve the appropriate fixture regardless of date prefix.
	// The Time column is expanded from HHMM to full ISO 8601 using the
	// fixture's date so the collector produces correct historical timestamps.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")

		var csvType string
		switch {
		case strings.HasSuffix(name, "_rpts_torn.csv"):
			csvType = "torn"
		case strings.HasSuffix(name, "_rpts_hail.csv"):
			csvType = "hail"
		case strings.HasSuffix(name, "_rpts_wind.csv"):
			csvType = "wind"
		default:
			http.NotFound(w, r)
			return
		}

		matches, _ := filepath.Glob(filepath.Join(dataDir, "*_rpts_"+csvType+".csv"))
		if len(matches) == 0 {
			log.Printf("no fixture found for type %s", csvType)
			http.Error(w, "fixture not found", http.StatusNotFound)
			return
		}

		data, err := os.ReadFile(matches[0])
		if err != nil {
			log.Printf("error reading %s: %v", matches[0], err)
			http.Error(w, "fixture not found", http.StatusInternalServerError)
			return
		}

		// Extract YYMMDD date from fixture filename (e.g. "240426_rpts_hail.csv")
		base := filepath.Base(matches[0])
		var fixtureDate time.Time
		if len(base) >= 6 {
			fixtureDate, _ = time.Parse("060102", base[:6])
		}

		// Expand HHMM times to ISO 8601 if we have a valid fixture date
		if !fixtureDate.IsZero() {
			data = expandTimes(data, fixtureDate)
		}

		log.Printf("serving %s for request %s", filepath.Base(matches[0]), r.URL.Path)
		w.Header().Set("Content-Type", "text/csv")
		if _, err := w.Write(data); err != nil {
			log.Printf("error writing response: %v", err)
		}
	})

	addr := ":" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("mock-server listening on %s (data_dir=%s)", addr, dataDir)
	log.Fatal(srv.ListenAndServe())
}

// expandTimes rewrites the Time column from HHMM to ISO 8601 using the given date.
// E.g. "1510" + 2024-04-26 → "2024-04-26T15:10:00Z"
func expandTimes(data []byte, date time.Time) []byte {
	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return data
	}

	// Find the Time column index
	header := records[0]
	timeIdx := -1
	for i, col := range header {
		if col == "Time" {
			timeIdx = i
			break
		}
	}
	if timeIdx < 0 {
		return data
	}

	dateStr := date.Format("2006-01-02")

	for i := 1; i < len(records); i++ {
		if timeIdx >= len(records[i]) {
			continue
		}
		hhmm := strings.TrimSpace(records[i][timeIdx])
		records[i][timeIdx] = expandHHMM(hhmm, dateStr)
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	_ = writer.WriteAll(records)
	writer.Flush()
	return buf.Bytes()
}

// expandHHMM converts an HHMM string to ISO 8601 with the given date prefix.
func expandHHMM(hhmm, dateStr string) string {
	if len(hhmm) < 3 {
		return dateStr + "T00:00:00Z"
	}
	padded := hhmm
	for len(padded) < 4 {
		padded = "0" + padded
	}
	hours := padded[:2]
	mins := padded[2:4]
	return dateStr + "T" + hours + ":" + mins + ":00Z"
}
