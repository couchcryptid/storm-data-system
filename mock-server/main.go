package main

import (
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
