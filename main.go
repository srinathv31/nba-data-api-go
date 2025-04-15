package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type TeamYear struct {
    Name     string `json:"name"`
    Year     string `json:"year"`
	Roster   string `json:"roster"`
	Schedule string `json:"schedule"`
}

type Roster struct {
    Name   string `json:"name"`
    Year   string `json:"year"`
    Roster string `json:"roster"`
}

type Schedule struct {
    Name     string `json:"name"`
    Year     string `json:"year"`
    Schedule string `json:"schedule"`
}

// responseWriter is a custom http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.ResponseWriter.WriteHeader(code)
		rw.wroteHeader = true
	}
}

// Initialize custom loggers
var (
	infoLogger  = log.New(os.Stdout, "\033[32mINFO: \033[0m", log.Ldate|log.Ltime)
	errorLogger = log.New(os.Stderr, "\033[31mERROR: \033[0m", log.Ldate|log.Ltime)
)

// loggingMiddleware logs information about each incoming request and its response
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := wrapResponseWriter(w)

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Calculate request duration
		duration := time.Since(start)

		// Ensure status is set, default to 200 if not set
		if wrapped.status == 0 {
			wrapped.status = 200
		}

		// Create the log entry
		logEntry := fmt.Sprintf(
			"%s %s (took %v)",
			r.Method,
			r.RequestURI,
			duration,
		)

		// Log based on status code
		if wrapped.status >= 400 {
			errorLogger.Printf("[%d] %s", wrapped.status, logEntry)
		} else {
			infoLogger.Printf("[%d] %s", wrapped.status, logEntry)
		}
	})
}

func main() {
	// Create a new router
	r := mux.NewRouter()

	// Apply the logging middleware to all routes
	r.Use(loggingMiddleware)

	// Define the route for the root URL
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to the NBA Data API!")
	})

	// Define the dynamic route for the team year endpoint
	r.HandleFunc("/v1/nba/{team}/{year}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		team := vars["team"]
		year := vars["year"]

		teamYear := TeamYear{
			Name: team,
			Year: year,
			Roster: "https://www.basketball-reference.com/teams/" + team + "/" + year + ".html",
			Schedule: "https://www.basketball-reference.com/teams/" + team + "/" + year + "_games.html",
		}

		json.NewEncoder(w).Encode(teamYear)
	}).Methods("GET")

	// Define the dynamic route for the team year roster endpoint
	r.HandleFunc("/v1/nba/{team}/{year}/roster", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		team := vars["team"]
		year := vars["year"]

		roster := Roster{
			Name: team,
			Year: year,
			Roster: "https://www.basketball-reference.com/teams/" + team + "/" + year + ".html",
		}

		json.NewEncoder(w).Encode(roster)
	}).Methods("GET")

	// Define the dynamic route for the team year schedule endpoint
	r.HandleFunc("/v1/nba/{team}/{year}/schedule", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		team := vars["team"]
		year := vars["year"]

		schedule := Schedule{
			Name: team,
			Year: year,
			Schedule: "https://www.basketball-reference.com/teams/" + team + "/" + year + "_games.html",
		}

		json.NewEncoder(w).Encode(schedule)
	}).Methods("GET")


	// Start the server
	infoLogger.Println("Starting server on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}