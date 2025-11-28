package geocoding

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/ngmaloney/marine-terminal/internal/database"
	_ "modernc.org/sqlite"
)

var (
	zipDB     *sql.DB
	zipDBOnce sync.Once
	initErr   error
)

// getZipcodeDB returns the singleton database connection
func getZipcodeDB(dbPath string) (*sql.DB, error) {
	zipDBOnce.Do(func() {
		// Provision database if it doesn't exist
		initErr = ProvisionZipcodeDatabase(dbPath)
		if initErr != nil {
			return
		}

		zipDB, initErr = sql.Open("sqlite", dbPath)
		if initErr != nil {
			return
		}

		// Set pragmas for performance
		_, initErr = zipDB.Exec(`
			PRAGMA journal_mode=WAL;
			PRAGMA synchronous=NORMAL;
			PRAGMA cache_size=10000;
		`)
	})
	return zipDB, initErr
}

// lookupZipcode looks up a zipcode in the SQLite database and returns a Location
func lookupZipcode(zipcode string) (*Location, error) {
	db, err := getZipcodeDB(database.DBPath())
	if err != nil {
		return nil, fmt.Errorf("opening zipcode database: %w", err)
	}
	return lookupZipcodeInDB(db, zipcode)
}

// lookupCityState looks up a city and state in the SQLite database and returns a Location
// If multiple zipcodes match, returns the first one (by zipcode)
func lookupCityState(city, state string) (*Location, error) {
	db, err := getZipcodeDB(database.DBPath())
	if err != nil {
		return nil, fmt.Errorf("opening zipcode database: %w", err)
	}
	return lookupCityStateInDB(db, city, state)
}

// lookupZipcodeInDB looks up a zipcode in the provided database connection
func lookupZipcodeInDB(db *sql.DB, zipcode string) (*Location, error) {
	var city, state string
	var lat, lon float64

	err := db.QueryRow(
		"SELECT city, state, latitude, longitude FROM zipcodes WHERE zipcode = ?",
		zipcode,
	).Scan(&city, &state, &lat, &lon)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("zipcode %s not found", zipcode)
	}
	if err != nil {
		return nil, fmt.Errorf("querying zipcode: %w", err)
	}

	return &Location{
		Latitude:  lat,
		Longitude: lon,
		Name:      fmt.Sprintf("%s, %s %s", city, state, zipcode),
	}, nil
}

// lookupCityStateInDB looks up a city and state in the provided database connection
func lookupCityStateInDB(db *sql.DB, city, state string) (*Location, error) {
	var zipcode, foundCity, foundState string
	var lat, lon float64

	err := db.QueryRow(
		"SELECT zipcode, city, state, latitude, longitude FROM zipcodes WHERE city = ? AND state = ? ORDER BY zipcode LIMIT 1",
		city, state,
	).Scan(&zipcode, &foundCity, &foundState, &lat, &lon)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no location found for %s, %s", city, state)
	}
	if err != nil {
		return nil, fmt.Errorf("querying city/state: %w", err)
	}

	return &Location{
		Latitude:  lat,
		Longitude: lon,
		Name:      fmt.Sprintf("%s, %s %s", foundCity, foundState, zipcode),
	}, nil
}
