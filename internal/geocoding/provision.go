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

const (
	zipcodeCSVURL = "https://raw.githubusercontent.com/midwire/free_zipcode_data/develop/all_us_zipcodes.csv"
)

// ProvisionZipcodeDatabase downloads and builds the zipcode table
func ProvisionZipcodeDatabase(dbPath string) error {
	// Check if table already exists
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Check if zipcodes table exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='zipcodes'").Scan(&count)
	if err != nil {
		return fmt.Errorf("checking for zipcodes table: %w", err)
	}
	if count > 0 {
		return nil // Table already exists
	}

	log.Println("Zipcode table not found, provisioning...")

	// Create data directory if it doesn't exist
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	// Download CSV
	csvPath := filepath.Join(dataDir, "all_us_zipcodes.csv")
	log.Printf("Downloading zipcode data from %s...\n", zipcodeCSVURL)
	if err := downloadZipcodeCSV(csvPath); err != nil {
		return fmt.Errorf("downloading zipcode CSV: %w", err)
	}
	defer os.Remove(csvPath) // Clean up after import

	// Build database
	log.Println("Building zipcode database...")
	if err := buildZipcodeDatabase(csvPath, dbPath); err != nil {
		return fmt.Errorf("building database: %w", err)
	}

	log.Printf("Successfully provisioned zipcode database at %s\n", dbPath)
	return nil
}

// downloadZipcodeCSV downloads the zipcode CSV file
func downloadZipcodeCSV(filepath string) error {
	resp, err := http.Get(zipcodeCSVURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// buildZipcodeDatabase creates a SQLite database from the CSV file
func buildZipcodeDatabase(csvPath, dbPath string) error {
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

	// Create index on state for efficient lookups
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_state ON zipcodes(state)`)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}

	// Open CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)

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

		// CSV format: Zipcode,ZipCodeType,City,State,LocationType,Lat,Long,...
		if len(record) < 7 {
			continue
		}

		zipcode := record[0]
		city := record[2]
		state := record[3]

		lat, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			continue
		}
		lon, err := strconv.ParseFloat(record[6], 64)
		if err != nil {
			continue
		}

		_, err = tx.Stmt(stmt).Exec(zipcode, city, state, lat, lon)
		if err != nil {
			continue
		}

		count++
		if count%5000 == 0 {
			log.Printf("Processed %d zipcodes...\n", count)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("Successfully created database with %d zipcodes\n", count)
	return nil
}
