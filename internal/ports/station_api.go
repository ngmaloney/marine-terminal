package ports

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ngmaloney/mariner-tui/internal/models"
)

const (
	stationAPIBaseURL = "https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi"
)

// zipCodeToCity maps ZIP codes to city names for station lookup
var zipCodeToCity = map[string]string{
	// Massachusetts - Cape Cod area
	"02633": "chatham", // Chatham
	"02634": "chatham",
	"02539": "woods hole", // Woods Hole
	"02554": "nantucket",  // Nantucket
	"02564": "nantucket",
	"02108": "boston", // Boston
	"02109": "boston",
	"02110": "boston",
	"02111": "boston",

	// Seattle area
	"98101": "seattle",
	"98102": "seattle",
	"98103": "seattle",
	"98104": "seattle",
	"98105": "seattle",
	"98106": "seattle",
	"98107": "seattle",
	"98108": "seattle",
	"98109": "seattle",
	"98110": "seattle",
	"98121": "seattle",
	"98122": "seattle",

	// San Francisco area
	"94102": "san francisco",
	"94103": "san francisco",
	"94104": "san francisco",
	"94105": "san francisco",
	"94107": "san francisco",
	"94108": "san francisco",
	"94109": "san francisco",
	"94110": "san francisco",
	"94111": "san francisco",
	"94112": "san francisco",

	// San Diego area
	"92101": "san diego",
	"92102": "san diego",
	"92103": "san diego",
	"92104": "san diego",
	"92105": "san diego",

	// New York area
	"10001": "new york",
	"10002": "new york",
	"10003": "new york",
	"10004": "new york",
	"10005": "new york",
	"10006": "new york",
	"10007": "new york",

	// Galveston, TX
	"77550": "galveston",
	"77551": "galveston",
	"77553": "galveston",

	// Panama City, FL
	"32401": "panama city",
	"32402": "panama city",
	"32403": "panama city",

	// Providence, RI
	"02901": "providence",
	"02902": "providence",
	"02903": "providence",
	"02904": "providence",

	// Bridgeport, CT
	"06601": "bridgeport",
	"06604": "bridgeport",
	"06605": "bridgeport",

	// Astoria, OR
	"97103": "astoria",

	// Wilmington, NC
	"28401": "wilmington",
	"28402": "wilmington",
	"28403": "wilmington",
	"28404": "wilmington",
	"28405": "wilmington",
}

// NOAAStationClient uses the NOAA Metadata API to search for stations
type NOAAStationClient struct {
	httpClient     *http.Client
	cache          map[string][]models.Port // Cache search results
	cacheMu        sync.RWMutex
	cacheTTL       time.Duration
	cacheTime      map[string]time.Time
	allStations    []models.Port // Cache of all stations
	stationsFetched bool
	stationsMu     sync.RWMutex
}

// NewNOAAStationClient creates a client that uses NOAA's Station Metadata API
func NewNOAAStationClient() *NOAAStationClient {
	return &NOAAStationClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cache:      make(map[string][]models.Port),
		cacheTime:  make(map[string]time.Time),
		cacheTTL:   24 * time.Hour, // Stations don't change often
	}
}

// stationResponse represents the NOAA API response
type stationResponse struct {
	Stations []struct {
		ID        string  `json:"id"`
		Name      string  `json:"name"`
		State     string  `json:"state"`
		Latitude  float64 `json:"lat"`
		Longitude float64 `json:"lng"`
		Type      string  `json:"type"` // R = reference, S = subordinate
	} `json:"stations"`
}

// SearchByLocation searches for stations using NOAA's API
func (c *NOAAStationClient) SearchByLocation(ctx context.Context, query string) ([]models.Port, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// Check cache first
	c.cacheMu.RLock()
	if cached, ok := c.cache[query]; ok {
		if time.Since(c.cacheTime[query]) < c.cacheTTL {
			c.cacheMu.RUnlock()
			return cached, nil
		}
	}
	c.cacheMu.RUnlock()

	// Check if query is a ZIP code and convert to city name
	originalQuery := query
	if cityName, ok := zipCodeToCity[query]; ok {
		query = cityName
	}

	// Determine search strategy
	var stations []models.Port
	var err error

	// Try state search first (most efficient)
	if len(query) == 2 {
		stations, err = c.searchByState(ctx, strings.ToUpper(query))
	} else {
		// Try to extract state abbreviation from query
		stations, err = c.searchByName(ctx, query)
	}

	if err != nil {
		return nil, err
	}

	if len(stations) == 0 {
		return nil, fmt.Errorf("no stations found for '%s'. Try: city name, state (MA, CA, WA), or ZIP code", query)
	}

	// Populate marine zones for results
	stations = PopulateMarineZones(ctx, stations)

	// Cache results using original query
	c.cacheMu.Lock()
	c.cache[originalQuery] = stations
	c.cacheTime[originalQuery] = time.Now()
	c.cacheMu.Unlock()

	return stations, nil
}

// searchByState searches for all stations in a given state
func (c *NOAAStationClient) searchByState(ctx context.Context, state string) ([]models.Port, error) {
	// Ensure we have all stations cached
	if err := c.ensureStationsCached(ctx); err != nil {
		return nil, err
	}

	// Filter by state
	c.stationsMu.RLock()
	defer c.stationsMu.RUnlock()

	var stations []models.Port
	for _, station := range c.allStations {
		if station.State == state {
			stations = append(stations, station)
		}
	}

	return stations, nil
}

// ensureStationsCached fetches all stations if not already cached
func (c *NOAAStationClient) ensureStationsCached(ctx context.Context) error {
	c.stationsMu.RLock()
	if c.stationsFetched {
		c.stationsMu.RUnlock()
		return nil
	}
	c.stationsMu.RUnlock()

	c.stationsMu.Lock()
	defer c.stationsMu.Unlock()

	// Double-check after acquiring write lock
	if c.stationsFetched {
		return nil
	}

	// Fetch all stations
	params := url.Values{}
	params.Add("type", "tidepredictions")

	stations, err := c.fetchStations(ctx, params)
	if err != nil {
		return fmt.Errorf("fetching all stations: %w", err)
	}

	c.allStations = stations
	c.stationsFetched = true

	return nil
}

// searchByName searches for stations by name/city
func (c *NOAAStationClient) searchByName(ctx context.Context, query string) ([]models.Port, error) {
	// Ensure we have all stations cached
	if err := c.ensureStationsCached(ctx); err != nil {
		return nil, err
	}

	// Search through cached stations
	c.stationsMu.RLock()
	defer c.stationsMu.RUnlock()

	var results []models.Port
	for _, station := range c.allStations {
		// Check if matches query
		if strings.Contains(strings.ToLower(station.Name), query) ||
			strings.Contains(strings.ToLower(station.City), query) {
			results = append(results, station)

			// Limit results to avoid overwhelming the user
			if len(results) >= 20 {
				break
			}
		}
	}

	return results, nil
}

// fetchStations makes the actual API call to NOAA
func (c *NOAAStationClient) fetchStations(ctx context.Context, params url.Values) ([]models.Port, error) {
	apiURL := fmt.Sprintf("%s/stations.json?%s", stationAPIBaseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching stations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NOAA API returned status %d", resp.StatusCode)
	}

	var stationResp stationResponse
	if err := json.NewDecoder(resp.Body).Decode(&stationResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Convert to Port models
	ports := make([]models.Port, 0, len(stationResp.Stations))
	for _, s := range stationResp.Stations {
		// Extract city from name (e.g., "Boston, MA" -> "Boston")
		city := s.Name
		if idx := strings.Index(city, ","); idx > 0 {
			city = city[:idx]
		}

		ports = append(ports, models.Port{
			ID:          s.ID,
			Name:        s.Name,
			City:        city,
			State:       s.State,
			Latitude:    s.Latitude,
			Longitude:   s.Longitude,
			TideStation: s.ID,
			Type:        "coastal",
		})
	}

	return ports, nil
}

// GetPortByID retrieves a specific port by station ID
func (c *NOAAStationClient) GetPortByID(ctx context.Context, stationID string) (*models.Port, error) {
	params := url.Values{}
	params.Add("type", "tidepredictions")
	params.Add("id", stationID)

	stations, err := c.fetchStations(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(stations) == 0 {
		return nil, fmt.Errorf("station not found: %s", stationID)
	}

	return &stations[0], nil
}
