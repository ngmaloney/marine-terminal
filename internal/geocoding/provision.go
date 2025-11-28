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

// ProvisionZipcodeDatabaseWithProgress downloads and builds the zipcode table with progress updates
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
	csvPath := filepath.Join(dataDir, "all_us_zipcodes.csv")
	sendProgress(fmt.Sprintf("Downloading zipcode data from %s...", zipcodeCSVURL))
	if err := downloadZipcodeCSV(csvPath); err != nil {
		return fmt.Errorf("downloading zipcode CSV: %w", err)
	}
	defer os.Remove(csvPath) // Clean up after import

	// Build database
	sendProgress("Building zipcode database...")
	if err := buildZipcodeDatabase(csvPath, dbPath, progressChan); err != nil {
		return fmt.Errorf("building database: %w", err)
	}

	sendProgress(fmt.Sprintf("Successfully provisioned zipcode database at %s", dbPath))
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
func buildZipcodeDatabase(csvPath, dbPath string, progressChan chan<- string) error {
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

		// CSV format: code,city,state,county,area_code,lat,lon
		if len(record) < 7 {
			continue
		}

		zipcode := record[0]
		city := record[1]
		state := record[2]

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
