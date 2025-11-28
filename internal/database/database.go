package database

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DBPath returns the path to the single shared database
func DBPath() string {
	return filepath.Join("data", "marine-terminal.db")
}

// EnsureUserSchema ensures that the user-specific tables (like user_ports) exist.
func EnsureUserSchema(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("opening database to ensure schema: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_ports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			state TEXT,
			city TEXT,
			zipcode TEXT,
			marine_zone_id TEXT NOT NULL,
			tide_station_id TEXT NOT NULL,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_ports_name ON user_ports(name);
	`)
	if err != nil {
		return fmt.Errorf("creating user_ports table: %w", err)
	}

	return nil
}
