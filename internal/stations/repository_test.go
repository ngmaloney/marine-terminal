package stations

import (
	"database/sql"
	"testing"

	"github.com/ngmaloney/marine-terminal/internal/database"
	_ "modernc.org/sqlite"
)

func TestFindNearbyStations(t *testing.T) {
	// Create an in-memory SQLite database for testing
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Create tide_stations table
	_, err = db.Exec(`
		CREATE TABLE tide_stations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			state TEXT,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL
		);
		CREATE INDEX idx_tide_stations_coords ON tide_stations(latitude, longitude);
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data (latitude, longitude)
	// Station A: Boston, MA (~42.36, -71.06)
	// Station B: New York, NY (~40.71, -74.01)
	// Station C: Miami, FL (~25.76, -80.19)
	_, err = db.Exec(`
		INSERT INTO tide_stations (id, name, state, latitude, longitude) VALUES
		('BOS', 'Boston Harbor', 'MA', 42.36, -71.06),
		('NYC', 'New York Harbor', 'NY', 40.71, -74.01),
		('MIA', 'Miami Beach', 'FL', 25.76, -80.19)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Mock GetDB to return our in-memory DB for this test
	oldGetDB := GetDB
	GetDB = func(dbPath string) (*sql.DB, error) { return db, nil }
	defer func() { GetDB = oldGetDB }()

	tests := []struct {
		name         string
		searchLat    float64
		searchLon    float64
		maxDistance  float64
		expectedID   string // Expect closest
		expectedErr  bool
	}{
		{"find nearest Boston", 42.36, -71.06, 5.0, "BOS", false},
		{"find nearest New York", 40.71, -74.01, 5.0, "NYC", false},
		{"no station within distance", 0.0, 0.0, 10.0, "", true},
		{"find within larger radius", 42.3, -71.0, 50.0, "BOS", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stations, err := FindNearbyStations(database.DBPath(), tt.searchLat, tt.searchLon, tt.maxDistance)

			if (err != nil) != tt.expectedErr {
				t.Errorf("FindNearbyStations() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			if !tt.expectedErr {
				if len(stations) == 0 {
					t.Error("FindNearbyStations() returned empty list")
					return
				}
				if stations[0].ID != tt.expectedID {
					t.Errorf("FindNearbyStations() closest ID = %v, want %v", stations[0].ID, tt.expectedID)
				}
				if stations[0].Distance > tt.maxDistance {
					t.Errorf("FindNearbyStations() returned station too far: %v > %v", stations[0].Distance, tt.maxDistance)
				}
			}
		})
	}
}

func TestGetStationByID(t *testing.T) {
	// Create an in-memory SQLite database for testing
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Create tide_stations table
	_, err = db.Exec(`
		CREATE TABLE tide_stations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			state TEXT,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO tide_stations (id, name, state, latitude, longitude) VALUES
		('TEST1', 'Test Station 1', 'CA', 34.05, -118.25)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Mock GetDB to return our in-memory DB for this test
	oldGetDB := GetDB
	GetDB = func(dbPath string) (*sql.DB, error) { return db, nil }
	defer func() { GetDB = oldGetDB }()

	// Test case: existing station
	station, err := GetStationByID(database.DBPath(), "TEST1")
	if err != nil {
		t.Errorf("GetStationByID() error = %v, want nil", err)
	}
	if station == nil || station.ID != "TEST1" || station.Name != "Test Station 1" {
		t.Errorf("GetStationByID() got %v, want TEST1", station)
	}

	// Test case: non-existent station
	_, err = GetStationByID(database.DBPath(), "NONEXISTENT")
	if err == nil {
		t.Error("GetStationByID() expected error for non-existent station, got nil")
	}
}
