package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Player struct {
    Name         string `json:"name" bson:"name"`
    RegularSeason struct {
        G    string `json:"G" bson:"G"`
        PER  string `json:"PER" bson:"PER"`
        TSP  string `json:"TS%" bson:"TS%"`
        WS   string `json:"WS" bson:"WS"`
    } `json:"regular_season" bson:"regular_season"`
    Playoffs struct {
        G    string `json:"G" bson:"G"`
        PER  string `json:"PER" bson:"PER"`
        TSP  string `json:"TS%" bson:"TS%"`
        WS   string `json:"WS" bson:"WS"`
    } `json:"playoffs" bson:"playoffs"`
}

type TeamYear struct {
    Team        string                 `json:"team" bson:"team"`
    FullName    string                 `json:"full_name" bson:"full_name"`
    Year        int                    `json:"year" bson:"year"`
    RosterURL   string                 `json:"roster_url" bson:"roster_url"`
    Roster      []Player      `json:"roster" bson:"roster"`
    ScheduleURL string                 `json:"schedule_url" bson:"schedule_url"`
    Schedule    map[string]interface{} `json:"schedule" bson:"schedule"`
}

type Roster struct {
    Team   string `json:"team"`
    Year   int `json:"year"`
    Roster string `json:"roster"`
}

type Schedule struct {
    Team     string `json:"team"`
    Year     int `json:"year"`
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
		yearStr := vars["year"]
		year, err := strconv.Atoi(yearStr)
		if err != nil {
			http.Error(w, "Invalid year format", http.StatusBadRequest)
			return
		}

		// find the team in the database
		collection := client.Database("nba-data").Collection("nba_seasons_v2")
		var result TeamYear
		err = collection.FindOne(context.TODO(), bson.M{"team": team, "year": year}).Decode(&result)
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
		yearStr := vars["year"]
		year, err := strconv.Atoi(yearStr)
		if err != nil {
			http.Error(w, "Invalid year format", http.StatusBadRequest)
			return
		}
		
		roster := Roster{
			Team: team,
			Year: year,
			Roster: "https://www.basketball-reference.com/teams/" + team + "/" + yearStr + ".html",
		}

		json.NewEncoder(w).Encode(roster)
	}).Methods("GET")

	// Define the dynamic route for the team year schedule endpoint
	r.HandleFunc("/v1/nba/{team}/{year}/schedule", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		team := vars["team"]
		yearStr := vars["year"]
		year, err := strconv.Atoi(yearStr)
		if err != nil {
			http.Error(w, "Invalid year format", http.StatusBadRequest)
			return
		}

		schedule := Schedule{
			Team: team,
			Year: year,
			Schedule: "https://www.basketball-reference.com/teams/" + team + "/" + yearStr + "_games.html",
		}

		json.NewEncoder(w).Encode(schedule)
	}).Methods("GET")


	// Start the server
	infoLogger.Println("Starting server on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}