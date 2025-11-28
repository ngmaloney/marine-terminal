// Package zonelookup provides marine zone lookups using SQLite database
package zonelookup

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	db   *sql.DB
	once sync.Once
	initErr error
)

// ZoneInfo represents a marine zone with its distance from a point
type ZoneInfo struct {
	Code     string
	Name     string
	Distance float64 // Distance in miles
}

// GetDB returns the singleton database connection
// Automatically provisions the database if it doesn't exist
func GetDB(dbPath string) (*sql.DB, error) {
	once.Do(func() {
		// Provision database if it doesn't exist
		initErr = ProvisionDatabase(dbPath)
		if initErr != nil {
			return
		}

		db, initErr = sql.Open("sqlite", dbPath)
		if initErr != nil {
			return
		}
		// Set pragmas for performance
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA synchronous=NORMAL")
		db.Exec("PRAGMA cache_size=10000")
	})
	return db, initErr
}

// haversineDistance calculates distance in miles between two lat/lon points
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMiles = 3959.0

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMiles * c
}

// GetNearbyMarineZones finds marine zones near the given coordinates
func GetNearbyMarineZones(dbPath string, lat, lon float64, maxDistanceMiles float64) ([]ZoneInfo, error) {
	db, err := GetDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	return getNearbyMarineZonesFromDB(db, lat, lon, maxDistanceMiles)
}

// getNearbyMarineZonesFromDB finds marine zones using the provided database connection
func getNearbyMarineZonesFromDB(db *sql.DB, lat, lon float64, maxDistanceMiles float64) ([]ZoneInfo, error) {
	// Query zones within an expanded bounding box (roughly +/- 1 degree = ~69 miles)
	// This is a rough filter to reduce the number of zones we calculate distance for
	latDelta := maxDistanceMiles / 69.0 * 1.5 // Add 50% margin
	lonDelta := maxDistanceMiles / 55.0 * 1.5 // Longitude degrees are smaller at higher latitudes

	query := `
		SELECT zone_code, zone_name, center_lat, center_lon
		FROM marine_zones
		WHERE center_lat BETWEEN ? AND ?
		  AND center_lon BETWEEN ? AND ?
	`

	rows, err := db.Query(query,
		lat-latDelta, lat+latDelta,
		lon-lonDelta, lon+lonDelta)
	if err != nil {
		return nil, fmt.Errorf("querying zones: %w", err)
	}
	defer rows.Close()

	var zones []ZoneInfo
	for rows.Next() {
		var code, name string
		var centerLat, centerLon float64

		if err := rows.Scan(&code, &name, &centerLat, &centerLon); err != nil {
			continue
		}

		// Calculate actual distance
		distance := HaversineDistance(lat, lon, centerLat, centerLon)

		// Only include zones within the max distance
		if distance <= maxDistanceMiles {
			zones = append(zones, ZoneInfo{
				Code:     code,
				Name:     name,
				Distance: distance,
			})
		}
	}

	// Sort by distance (closest first)
	sort.Slice(zones, func(i, j int) bool {
		return zones[i].Distance < zones[j].Distance
	})

	return zones, nil
}

// getZoneInfoByCodeFromDB retrieves a single marine zone using the provided database connection
func getZoneInfoByCodeFromDB(db *sql.DB, zoneCode string) (*ZoneInfo, error) {
	var code, name string
	var centerLat, centerLon float64

	err := db.QueryRow(
		"SELECT zone_code, zone_name, center_lat, center_lon FROM marine_zones WHERE zone_code = ?",
		zoneCode,
	).Scan(&code, &name, &centerLat, &centerLon)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("zone code %s not found", zoneCode)
	}
	if err != nil {
		return nil, fmt.Errorf("querying zone by code: %w", err)
	}

	// For direct lookup, distance is not relevant, so set to 0.0
	return &ZoneInfo{
		Code:     code,
		Name:     name,
		Distance: 0.0,
	}, nil
}
