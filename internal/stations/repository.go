package stations

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/ngmaloney/marine-terminal/internal/zonelookup"
	_ "modernc.org/sqlite"
)

// TideStationInfo represents a tide station with its distance from a point
type TideStationInfo struct {
	ID        string
	Name      string
	State     string
	Latitude  float64
	Longitude float64
	Distance  float64 // Distance in miles
}

var (
	db   *sql.DB
	once sync.Once
	initErr error

	// GetDB is a function variable to allow mocking in tests
	GetDB = func(dbPath string) (*sql.DB, error) {
		once.Do(func() {
			// Provision database if it doesn't exist
			initErr = ProvisionStationsDatabase(dbPath, nil)
			if initErr != nil {
				return
			}

			db, initErr = sql.Open("sqlite", dbPath)
			if initErr != nil {
				return
			}
			// Set pragmas for performance
			_, _ = db.Exec("PRAGMA journal_mode=WAL")
			_, _ = db.Exec("PRAGMA synchronous=NORMAL")
			_, _ = db.Exec("PRAGMA cache_size=10000")
		})
		return db, initErr
	}
)

// FindNearbyStations finds tide stations near the given coordinates within a max distance.
func FindNearbyStations(dbPath string, lat, lon float64, maxDistanceMiles float64) ([]TideStationInfo, error) {
	db, err := GetDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Use a bounding box to initially filter stations for performance.
	// A rough estimate: 1 degree of lat/lon is approximately 69 miles.
	// We add a margin (e.g., 1.5x maxDistanceMiles) to ensure we don't miss stations near the edge.
	latDelta := (maxDistanceMiles / 69.0) * 1.5
	lonDelta := (maxDistanceMiles / (69.0 * math.Cos(lat*math.Pi/180))) * 1.5 // Adjust for longitude at higher latitudes

	query := `
		SELECT id, name, state, latitude, longitude
		FROM tide_stations
		WHERE latitude BETWEEN ? AND ?
		  AND longitude BETWEEN ? AND ?
	`

	rows, err := db.Query(query,
		lat-latDelta, lat+latDelta,
		lon-lonDelta, lon+lonDelta)
	if err != nil {
		return nil, fmt.Errorf("querying stations: %w", err)
	}
	defer rows.Close()

	var potentialStations []TideStationInfo
	for rows.Next() {
		var id, name, state string
		var stationLat, stationLon float64

		if err := rows.Scan(&id, &name, &state, &stationLat, &stationLon); err != nil {
			continue
		}

		// Calculate actual distance using Haversine formula
		distance := zonelookup.HaversineDistance(lat, lon, stationLat, stationLon)

		// Only consider stations within the specified maxDistanceMiles
		if distance <= maxDistanceMiles {
			potentialStations = append(potentialStations, TideStationInfo{
				ID:        id,
				Name:      name,
				State:     state,
				Latitude:  stationLat,
				Longitude: stationLon,
				Distance:  distance,
			})
		}
	}

	if len(potentialStations) == 0 {
		return nil, fmt.Errorf("no tide stations found near %.4f, %.4f within %.1f miles", lat, lon, maxDistanceMiles)
	}

	// Sort by distance to find the nearest
	sort.Slice(potentialStations, func(i, j int) bool {
		return potentialStations[i].Distance < potentialStations[j].Distance
	})

	return potentialStations, nil
}

// GetStationByID retrieves a single tide station by its ID.
func GetStationByID(dbPath, stationID string) (*TideStationInfo, error) {
	db, err := GetDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	var id, name, state string
	var lat, lon float64

	err = db.QueryRow(
		"SELECT id, name, state, latitude, longitude FROM tide_stations WHERE id = ?",
		stationID,
	).Scan(&id, &name, &state, &lat, &lon)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tide station %s not found", stationID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying tide station by ID: %w", err)
	}

	return &TideStationInfo{
		ID:        id,
		Name:      name,
		State:     state,
		Latitude:  lat,
		Longitude: lon,
		Distance:  0.0, // Distance not relevant for direct ID lookup
	}, nil
}
