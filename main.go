package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"time"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

type Event struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description, omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	CreatedAt   time.Time `json:"created_at"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("postgres", "host=localhost port=5432 user=postgres dbname=events sslmode=disable password=<PASSWORD>")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	http.HandleFunc("/events/", eventsByIdHandler)
	http.HandleFunc("/events", eventsHandler)
	log.Println("Server started on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listEvents(w, r)
	case http.MethodPost:
		createEvent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func eventsByIdHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Path[len("/events/"):]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	getEventByID(w, r, id)
}

func createEvent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title       string    `json:"title"`
		Description string    `json:"description"`
		StartTime   time.Time `json:"start_time"`
		EndTime     time.Time `json:"end_time"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := validateInput(in.Title, in.StartTime, in.EndTime); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	e := Event{
		ID:          uuid.New(),
		Title:       in.Title,
		Description: in.Description,
		StartTime:   in.StartTime,
		EndTime:     in.EndTime,
		CreatedAt:   time.Now(),
	}

	query := `INSERT INTO events (id, title, description, start_time, end_time, created_at) VALUES 
                    ($1, $2, $3, $4, $5, $6) 
                     RETURNING id, title, description, start_time, end_time, created_at;
	`

	ctx := r.Context()
	row := db.QueryRowContext(ctx, query, e.ID, e.Title, e.Description, e.StartTime, e.EndTime, e.CreatedAt)
	if err := row.Scan(&e.ID, &e.Title, &e.Description, &e.StartTime, &e.EndTime, &e.CreatedAt); err != nil {
		http.Error(w, "Error inserting event", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(e)
}

func listEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := db.QueryContext(ctx, "SELECT id, title, description, start_time, end_time, created_at FROM events ORDER BY start_time ASC")
	if err != nil {
		http.Error(w, "Error fetching events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.StartTime, &e.EndTime, &e.CreatedAt); err != nil {
			http.Error(w, "Error scanning event", http.StatusInternalServerError)
			return
		}
		events = append(events, e)
	}
	json.NewEncoder(w).Encode(events)
}

func getEventByID(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	row := db.QueryRowContext(ctx, "SELECT id, title, description, start_time, end_time, created_at FROM events WHERE id = $1", id)
	var e Event
	err := row.Scan(&e.ID, &e.Title, &e.Description, &e.StartTime, &e.EndTime, &e.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "db Error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(e)
}

func validateInput(title string, startTime time.Time, endTime time.Time) error {
	if title == "" {
		return errors.New("title is required")
	}
	if len(title) > 100 {
		return errors.New("title too long")
	}

	if startTime.After(endTime) {
		return errors.New("start time must be before end time")
	}
	return nil
}
