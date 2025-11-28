package stations

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

// Reset the singleton DB and provisionMu for testing
func resetSingletons() {
	provisionMu = sync.Mutex{}
}

func TestNeedsProvisioning(t *testing.T) {
	resetSingletons()
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Case 1: DB file does not exist
	needs, err := NeedsProvisioning(dbPath)
	if err != nil || !needs {
		t.Errorf("Case 1: Expected needs=true, err=nil; got needs=%v, err=%v", needs, err)
	}

	// Case 2: DB file exists, but table does not (create a valid, empty sqlite db file)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to create empty db file: %v", err)
	}
	db.Close()
	needs, err = NeedsProvisioning(dbPath)
	if err != nil || !needs {
		t.Errorf("Case 2: Expected needs=true, err=nil; got needs=%v, err=%v", needs, err)
	}

	// Case 3: DB file exists and table exists
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE tide_stations (id TEXT PRIMARY KEY);`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	db.Close()

	needs, err = NeedsProvisioning(dbPath)
	if err != nil || needs {
		t.Errorf("Case 3: Expected needs=false, err=nil; got needs=%v, err=%v", needs, err)
	}
}

func TestFetchAllTideStations(t *testing.T) {
	resetSingletons()
	// Mock NOAA API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stations.json" || r.URL.Query().Get("type") != "tidepredictions" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"stations": [{"id": "1", "name": "Station 1", "lat": "10.0", "lng": "-10.0", "state": "MA"}]}`)
	}))
	defer server.Close()

	// Temporarily override stationAPIBaseURL for testing
	oldURL := stationAPIBaseURL
	stationAPIBaseURL = server.URL
	defer func() { stationAPIBaseURL = oldURL }()

	stations, err := fetchAllTideStations(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(stations) != 1 || stations[0].ID != "1" {
		t.Errorf("Expected 1 station with ID \"1\", got %v", stations)
	}
}

func TestBuildStationsDatabase(t *testing.T) {
	resetSingletons()
	// Create an in-memory SQLite database for testing
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	stations := []Station{
		{ID: "1", Name: "Station One", State: "MA", Latitude: "10.0", Longitude: "-10.0"},
		{ID: "2", Name: "Station Two", State: "CA", Latitude: "20.0", Longitude: "-20.0"},
	}

	err = buildStationsDatabase(db, stations, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify data was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tide_stations").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query count: %v", err)
	}
	if count != len(stations) {
		t.Errorf("Expected %d stations, got %d", len(stations), count)
	}

	var name string
	err = db.QueryRow("SELECT name FROM tide_stations WHERE id = ?", "1").Scan(&name)
	if err != nil || name != "Station One" {
		t.Errorf("Expected name \"Station One\" for ID \"1\", got %v", name)
	}
}

func TestProvisionStationsDatabase(t *testing.T) {
	resetSingletons()
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "marine-terminal.db")

	// Mock NOAA API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stations.json" || r.URL.Query().Get("type") != "tidepredictions" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"stations": [{"id": "TEST1", "name": "Test Station", "lat": "1.0", "lng": "-1.0", "state": "TX"}]}`)
	}))
	defer server.Close()

	// Temporarily override stationAPIBaseURL for testing
	oldURL := stationAPIBaseURL
	stationAPIBaseURL = server.URL
	defer func() { stationAPIBaseURL = oldURL }()

	// Run provisioning
	err := ProvisionStationsDatabase(dbPath, nil)
	if err != nil {
		t.Fatalf("ProvisionStationsDatabase() error = %v", err)
	}

	// Verify table and data exist
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open db after provisioning: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tide_stations").Scan(&count)
	if err != nil || count != 1 {
		t.Errorf("Expected 1 station in DB, got %d (err: %v)", count, err)
	}

	var name string
	err = db.QueryRow("SELECT name FROM tide_stations WHERE id = 'TEST1'").Scan(&name)
	if err != nil || name != "Test Station" {
		t.Errorf("Expected station 'Test Station', got %v (err: %v)", name, err)
	}

	// Second call should not re-provision
	count = 0
	err = db.QueryRow("SELECT COUNT(*) FROM tide_stations").Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("Expected 1 station before re-provisioning test, got %d (err: %v)", count, err)
	}

	err = ProvisionStationsDatabase(dbPath, nil)
	if err != nil {
		t.Fatalf("Second ProvisionStationsDatabase() error = %v", err)
	}
	// Count should still be 1 because it didn't re-provision due to `INSERT OR IGNORE` and `NeedsProvisioning`
	err = db.QueryRow("SELECT COUNT(*) FROM tide_stations").Scan(&count)
	if err != nil || count != 1 {
		t.Errorf("Expected 1 station after second provisioning call, got %d (err: %v)", count, err)
	}
}
