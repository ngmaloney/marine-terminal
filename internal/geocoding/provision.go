package geocoding

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	_ "modernc.org/sqlite"
)

// NeedsProvisioning checks if the zipcode database needs to be provisioned
func NeedsProvisioning(dbPath string) (bool, error) {
	// If file doesn't exist, we need to provision
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return true, nil
	}

	// Check if zipcodes table exists
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return false, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='zipcodes'").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking for zipcodes table: %w", err)
	}

	return count == 0, nil
}

// ProvisionZipcodeDatabase downloads and builds the zipcode table
func ProvisionZipcodeDatabase(dbPath string) error {
	return ProvisionZipcodeDatabaseWithProgress(dbPath, nil)
}

// ProvisionZipcodeDatabaseWithProgress builds the zipcode table from downloaded CSV data
func ProvisionZipcodeDatabaseWithProgress(dbPath string, progressChan chan<- string) error {
	needs, err := NeedsProvisioning(dbPath)
	if err != nil {
		return err
	}
	if !needs {
		return nil
	}

	sendProgress := func(msg string) {
		if progressChan != nil {
			progressChan <- msg
		} else {
			log.Println(msg)
		}
	}

	sendProgress("Zipcode table not found, provisioning...")

	// Create data directory if it doesn't exist
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	// Download CSV
	url := "https://raw.githubusercontent.com/ngmaloney/marine-terminal/5109f78ca2f1a928ad0d241f93ed51e0306fa54d/testdata/uszips.csv"
	sendProgress(fmt.Sprintf("Downloading zipcode data from %s...", url))

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading zipcode data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Build database from downloaded CSV
	sendProgress("Building zipcode database from downloaded data...")
	if err := buildZipcodeDatabase(resp.Body, dbPath, progressChan); err != nil {
		return fmt.Errorf("building database: %w", err)
	}

	sendProgress(fmt.Sprintf("Successfully provisioned zipcode database at %s", dbPath))
	return nil
}

// buildZipcodeDatabase creates a SQLite database from the CSV data
func buildZipcodeDatabase(csvData io.Reader, dbPath string, progressChan chan<- string) error {
	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS zipcodes (
			zipcode TEXT PRIMARY KEY,
			city TEXT NOT NULL,
			state TEXT NOT NULL,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	// Create indexes for efficient lookups
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_state ON zipcodes(state);
		CREATE INDEX IF NOT EXISTS idx_city_state ON zipcodes(city, state);
	`)
	if err != nil {
		return fmt.Errorf("creating indexes: %w", err)
	}

	reader := csv.NewReader(csvData)

	// Skip header
	_, err = reader.Read()
	if err != nil {
		return err
	}

	// Prepare insert statement
	stmt, err := db.Prepare("INSERT OR IGNORE INTO zipcodes (zipcode, city, state, latitude, longitude) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Begin transaction for faster inserts
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip invalid records
		}

		// SimpleMaps CSV format: zip,lat,lng,city,state_id,state_name,...
		if len(record) < 6 {
			continue
		}

		zipcode := record[0]
		city := record[3]
		state := record[4] // state_id (e.g., "MA")

		lat, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			continue
		}
		lon, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			continue
		}

		_, err = tx.Stmt(stmt).Exec(zipcode, city, state, lat, lon)
		if err != nil {
			continue		}

		count++
		if count%5000 == 0 {
			msg := fmt.Sprintf("Processed %d zipcodes...", count)
			if progressChan != nil {
				progressChan <- msg
			} else {
				log.Println(msg)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	msg := fmt.Sprintf("Successfully created database with %d zipcodes", count)
	if progressChan != nil {
		progressChan <- msg
	} else {
		log.Println(msg)
	}
	return nil
}