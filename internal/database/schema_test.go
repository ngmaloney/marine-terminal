package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureUserSchema_Persistence(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "marine_terminal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// 1. Initialize schema
	if err := EnsureUserSchema(dbPath); err != nil {
		t.Fatalf("First EnsureUserSchema failed: %v", err)
	}

	// 2. Insert a record
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	_, err = db.Exec(`INSERT INTO user_ports (name, marine_zone_id, tide_station_id, latitude, longitude) VALUES ('Test Port', 'Z1', 'S1', 0.0, 0.0)`)
	db.Close()
	if err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	// 3. Initialize schema again (should not drop table)
	if err := EnsureUserSchema(dbPath); err != nil {
		t.Fatalf("Second EnsureUserSchema failed: %v", err)
	}

	// 4. Verify record exists
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM user_ports WHERE name = 'Test Port'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query record: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 record, got %d. Data was likely lost due to table drop.", count)
	}
}
