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
		VALUES ('12345', 'Test City', 'TS', 40.7128, -74.0060)
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
