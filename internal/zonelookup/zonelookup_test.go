package zonelookup

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestGetNearbyMarineZonesFromDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE marine_zones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			zone_code TEXT NOT NULL,
			zone_name TEXT,
			center_lat REAL NOT NULL,
			center_lon REAL NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test zones
	// Point A: 40.0, -70.0
	// Point B (Near A): 40.1, -70.1
	// Point C (Far from A): 50.0, -80.0
	_, err = db.Exec(`
		INSERT INTO marine_zones (zone_code, zone_name, center_lat, center_lon) VALUES
		('Z1', 'Near Zone', 40.1, -70.1),
		('Z2', 'Far Zone', 50.0, -80.0)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	tests := []struct {
		name       string
		lat        float64
		lon        float64
		maxDist    float64
		wantZones  int
		targetZone string
	}{
		{"find near zone", 40.0, -70.0, 50.0, 1, "Z1"},
		{"too far", 40.0, -70.0, 5.0, 0, ""},
		{"find nothing", 0.0, 0.0, 100.0, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zones, err := getNearbyMarineZonesFromDB(db, tt.lat, tt.lon, tt.maxDist)
			if err != nil {
				t.Fatalf("getNearbyMarineZonesFromDB() error = %v", err)
			}
			if len(zones) != tt.wantZones {
				t.Errorf("got %d zones, want %d", len(zones), tt.wantZones)
			}
			if tt.wantZones > 0 {
				if zones[0].Code != tt.targetZone {
					t.Errorf("got zone %s, want %s", zones[0].Code, tt.targetZone)
				}
			}
		})
	}
}

func TestGetZoneInfoByCodeFromDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE marine_zones (
			zone_code TEXT NOT NULL,
			zone_name TEXT,
			center_lat REAL NOT NULL,
			center_lon REAL NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO marine_zones (zone_code, zone_name, center_lat, center_lon) VALUES
		('Z1', 'Test Zone', 40.0, -70.0)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test found
	zone, err := getZoneInfoByCodeFromDB(db, "Z1")
	if err != nil {
		t.Errorf("getZoneInfoByCodeFromDB('Z1') error = %v", err)
	}
	if zone == nil || zone.Name != "Test Zone" {
		t.Errorf("getZoneInfoByCodeFromDB('Z1') = %v, want 'Test Zone'", zone)
	}

	// Test not found
	_, err = getZoneInfoByCodeFromDB(db, "Z999")
	if err == nil {
		t.Error("getZoneInfoByCodeFromDB('Z999') expected error, got nil")
	}
}
