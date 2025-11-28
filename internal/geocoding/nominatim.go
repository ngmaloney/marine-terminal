package geocoding

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	nominatimURL = "https://nominatim.openstreetmap.org/search"
	userAgent    = "MarineTerminal/1.0" // Required by Nominatim ToS
)

// Geocoder converts addresses to coordinates
type Geocoder struct {
	httpClient *http.Client
	lastCall   time.Time
	mu         sync.Mutex
}

// Location represents a geocoded location
type Location struct {
	Latitude  float64
	Longitude float64
	Name      string
}

// NewGeocoder creates a new geocoder
func NewGeocoder() *Geocoder {
	return &Geocoder{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// nominatimResponse represents the Nominatim API response
type nominatimResponse struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
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

	// For city/state queries, use Nominatim API
	// Build query parameters for Nominatim
	params := url.Values{}
	params.Add("format", "json")
	params.Add("limit", "1")

	// Treat as city/state or general address
	params.Add("q", query+", USA")

	// Build URL
	reqURL := fmt.Sprintf("%s?%s", nominatimURL, params.Encode())

	// Rate limiting: Nominatim requires 1 req/sec max
	g.mu.Lock()
	if !g.lastCall.IsZero() {
		elapsed := time.Since(g.lastCall)
		if elapsed < time.Second {
			time.Sleep(time.Second - elapsed)
		}
	}
	g.lastCall = time.Now()
	g.mu.Unlock()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set required User-Agent header (Nominatim ToS requirement)
	req.Header.Set("User-Agent", userAgent)

	// Execute request
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nominatim API returned status %d", resp.StatusCode)
	}

	// Parse response
	var results []nominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for '%s'", query)
	}

	result := results[0]

	// Parse coordinates
	var lat, lon float64
	if _, err := fmt.Sscanf(result.Lat, "%f", &lat); err != nil {
		return nil, fmt.Errorf("parsing latitude: %w", err)
	}
	if _, err := fmt.Sscanf(result.Lon, "%f", &lon); err != nil {
		return nil, fmt.Errorf("parsing longitude: %w", err)
	}

	return &Location{
		Latitude:  lat,
		Longitude: lon,
		Name:      result.DisplayName,
	}, nil
}

// isZipcode checks if a string looks like a US zipcode
func isZipcode(s string) bool {
	// Match 5-digit or 9-digit (with hyphen) zipcodes
	matched, _ := regexp.MatchString(`^\d{5}(-\d{4})?$`, s)
	return matched
}
