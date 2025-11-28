package geocoding

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// Geocoder converts addresses to coordinates using local database
type Geocoder struct{}

// Location represents a geocoded location
type Location struct {
	Latitude  float64
	Longitude float64
	Name      string
}

// NewGeocoder creates a new geocoder
func NewGeocoder() *Geocoder {
	return &Geocoder{}
}

// Geocode converts a query (zipcode, city/state, etc.) to coordinates
func (g *Geocoder) Geocode(ctx context.Context, query string) (*Location, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Check if query looks like a zipcode - use SQLite database
	if isZipcode(query) {
		return lookupZipcode(query)
	}

	// For city/state queries, parse and use local database
	// Expected format: "City, ST" or "City, State"
	parts := strings.Split(query, ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid format: expected 'City, State' (e.g., 'Chatham, MA')")
	}

	city := strings.TrimSpace(parts[0])
	state := strings.TrimSpace(parts[1])

	if city == "" || state == "" {
		return nil, fmt.Errorf("city and state cannot be empty")
	}

	// Look up in local database
	return lookupCityState(city, state)
}

// isZipcode checks if a string looks like a US zipcode
func isZipcode(s string) bool {
	// Match 5-digit or 9-digit (with hyphen) zipcodes
	matched, _ := regexp.MatchString(`^\d{5}(-\d{4})?$`, s)
	return matched
}
