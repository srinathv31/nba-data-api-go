package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type TeamYear struct {
    Name     string `json:"name" bson:"name"`
	FullName string `json:"full_name" bson:"full_name"`
    Year     string `json:"year" bson:"year"`
	Roster   string `json:"roster" bson:"roster"`
	Schedule string `json:"schedule" bson:"schedule"`
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
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Create a new router
	r := mux.NewRouter()

	// connect to mongo db
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("Missing 'MONGODB_URI' environment variable.")
	}
	client, err := mongo.Connect(options.Client().
		ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

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

		// find the team in the database
		collection := client.Database("nba-data").Collection("teams")
		var result TeamYear
		err := collection.FindOne(context.TODO(), bson.M{"name": team, "year": year}).Decode(&result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(result)
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