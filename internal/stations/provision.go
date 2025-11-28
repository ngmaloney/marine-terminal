package stations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	stationAPIBaseURL = "https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi"
	provisionMu sync.Mutex
)

// Station represents a NOAA tide station
type Station struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	State     string  `json:"state"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

// stationResponse represents the NOAA MDAPI response for multiple stations
type stationResponse struct {
	Stations []Station `json:"stations"`
}

// NeedsProvisioning checks if the tide stations database needs to be provisioned
func NeedsProvisioning(dbPath string) (bool, error) {
	// If file doesn't exist, we need to provision
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return true, nil
	}

	// Check if tide_stations table exists
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return false, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tide_stations'").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking for tide_stations table: %w", err)
	}

	return count == 0, nil
}

// ProvisionStationsDatabase fetches all active tide stations from NOAA and stores them in the SQLite database
func ProvisionStationsDatabase(dbPath string, progressChan chan<- string) error {
	provisionMu.Lock()
	defer provisionMu.Unlock()

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

	sendProgress("Tide stations table not found, provisioning...")

	// Create data directory if it doesn't exist
	dataDir := filepath.Dir(dbPath)
	if err = os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	// Fetch all tide stations
	sendProgress(fmt.Sprintf("Downloading tide station data from %s...", stationAPIBaseURL))
	stations, err := fetchAllTideStations(context.Background())
	if err != nil {
		return fmt.Errorf("fetching all tide stations: %w", err)
	}

	// Open database (or create if it doesn't exist)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("opening database for building: %w", err)
	}
	defer db.Close()

	// Build database
	sendProgress("Building tide stations database...")
	if err = buildStationsDatabase(db, stations, progressChan); err != nil {
		return fmt.Errorf("building database: %w", err)
	}

	sendProgress(fmt.Sprintf("Successfully provisioned tide stations database at %s", dbPath))
	return nil
}

// fetchAllTideStations fetches all active tide stations from the NOAA MDAPI
func fetchAllTideStations(ctx context.Context) ([]Station, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	apiURL := fmt.Sprintf("%s/stations.json?type=tidepredictions", stationAPIBaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching stations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NOAA MDAPI returned status %d", resp.StatusCode)
	}

	var stationResp stationResponse
	if err := json.NewDecoder(resp.Body).Decode(&stationResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return stationResp.Stations, nil
}

// buildStationsDatabase creates the tide_stations table and inserts fetched stations
func buildStationsDatabase(db *sql.DB, stations []Station, progressChan chan<- string) error {
	var err error

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tide_stations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			state TEXT,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tide_stations_coords ON tide_stations(latitude, longitude);
		CREATE INDEX IF NOT EXISTS idx_tide_stations_state ON tide_stations(state);
	`)
	if err != nil {
		return fmt.Errorf("creating tide_stations table: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback on error

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO tide_stations (id, name, state, latitude, longitude) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	count := 0
	for _, s := range stations {
		_, err = stmt.Exec(s.ID, s.Name, s.State, s.Latitude, s.Longitude)
		if err != nil {
			log.Printf("Error inserting station %s: %v", s.ID, err)
			continue
		}
		count++
		if count%500 == 0 {
			if progressChan != nil {
				progressChan <- fmt.Sprintf("Inserted %d tide stations...", count)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	if progressChan != nil {
		progressChan <- fmt.Sprintf("Successfully inserted %d tide stations", count)
	}
	return nil
}

