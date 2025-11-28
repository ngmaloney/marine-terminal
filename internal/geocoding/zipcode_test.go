package geocoding

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestLookupZipcodeInDB(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Create table
	_, err = db.Exec(`
		CREATE TABLE zipcodes (
			zipcode TEXT PRIMARY KEY,
			city TEXT NOT NULL,
			state TEXT NOT NULL,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO zipcodes (zipcode, city, state, latitude, longitude)
		VALUES
			('12345', 'Test City', 'TS', 40.7128, -74.0060),
			('02633', 'Chatham', 'MA', 41.6885, -69.9511)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	tests := []struct {
		name    string
		zipcode string
		want    bool // whether we expect a result
	}{
		{"existing zipcode", "12345", true},
		{"non-existent zipcode", "99999", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := lookupZipcodeInDB(db, tt.zipcode)
			if tt.want {
				if err != nil {
					t.Errorf("lookupZipcodeInDB() error = %v, want nil", err)
					return
				}
				if loc == nil {
					t.Error("lookupZipcodeInDB() returned nil location")
					return
				}
				if loc.Name != "Test City, TS 12345" {
					t.Errorf("lookupZipcodeInDB() name = %v, want 'Test City, TS 12345'", loc.Name)
				}
			} else {
				if err == nil {
					t.Error("lookupZipcodeInDB() expected error, got nil")
				}
			}
		})
	}
}

func TestLookupCityStateInDB(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Create table
	_, err = db.Exec(`
		CREATE TABLE zipcodes (
			zipcode TEXT PRIMARY KEY,
			city TEXT NOT NULL,
			state TEXT NOT NULL,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO zipcodes (zipcode, city, state, latitude, longitude)
		VALUES
			('02633', 'Chatham', 'MA', 41.6885, -69.9511),
			('98101', 'Seattle', 'WA', 47.6062, -122.3321)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	tests := []struct {
		name      string
		city      string
		state     string
		wantFound bool
		wantName  string
	}{
		{"existing city/state Chatham", "Chatham", "MA", true, "Chatham, MA 02633"},
		{"existing city/state Seattle", "Seattle", "WA", true, "Seattle, WA 98101"},
		{"non-existent city", "Nowhere", "XX", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := lookupCityStateInDB(db, tt.city, tt.state)
			if tt.wantFound {
				if err != nil {
					t.Errorf("lookupCityStateInDB() error = %v, want nil", err)
					return
				}
				if loc == nil {
					t.Error("lookupCityStateInDB() returned nil location")
					return
				}
				if loc.Name != tt.wantName {
					t.Errorf("lookupCityStateInDB() name = %v, want %v", loc.Name, tt.wantName)
				}
			} else {
				if err == nil {
					t.Error("lookupCityStateInDB() expected error, got nil")
				}
			}
		})
	}
}
