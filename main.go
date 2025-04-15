package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

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
		fmt.Fprintf(w, "You've requested the team %s in the year %s\n", team, year)
	}).Methods("GET")

	// Define the dynamic route for the team year roster endpoint
	r.HandleFunc("/v1/nba/{team}/{year}/roster", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		team := vars["team"]
		year := vars["year"]
		fmt.Fprintf(w, "You've requested the roster for the team %s in the year %s\n", team, year)
	}).Methods("GET")

	// Define the dynamic route for the team year schedule endpoint
	r.HandleFunc("/v1/nba/{team}/{year}/schedule", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		team := vars["team"]
		year := vars["year"]
		fmt.Fprintf(w, "You've requested the schedule for the team %s in the year %s\n", team, year)
	}).Methods("GET")


	// Start the server
	infoLogger.Println("Starting server on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}