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

	"github.com/ngmaloney/marine-terminal/internal/models"
)

const (
	stationAPIBaseURL = "https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi"
)

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

	// Determine search strategy
	var stations []models.Port
	var err error

	// Try state search first (most efficient)
	if len(query) == 2 {
		stations, err = c.searchByState(ctx, strings.ToUpper(query))
	} else {
		// Search by name/city
		stations, err = c.searchByName(ctx, query)
	}

	if err != nil {
		return nil, err
	}

	// If no exact matches, try to find nearby stations for known marine areas
	if len(stations) == 0 {
		stations = c.findNearbyMarineStations(ctx, query)
	}

	if len(stations) == 0 {
		return nil, fmt.Errorf("no stations found for '%s'. Try: city name or state abbreviation (MA, CA, WA)", query)
	}

	// Populate marine zones for results
	stations = PopulateMarineZones(ctx, stations)

	// Cache results
	c.cacheMu.Lock()
	c.cache[query] = stations
	c.cacheTime[query] = time.Now()
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

	// Parse query to extract city and optional state
	// Format: "City" or "City, State" or "City State" or "City,State"
	var cityQuery, stateQuery string

	// Try to split by comma first
	if idx := strings.Index(query, ","); idx > 0 {
		cityQuery = strings.TrimSpace(query[:idx])
		stateQuery = strings.TrimSpace(query[idx+1:])
		stateQuery = strings.ToUpper(stateQuery) // States are uppercase
	} else {
		// Try to split by space and check if last part is a state abbreviation
		parts := strings.Fields(query)
		if len(parts) >= 2 {
			lastPart := strings.ToUpper(parts[len(parts)-1])
			if len(lastPart) == 2 {
				// Likely a state abbreviation
				stateQuery = lastPart
				cityQuery = strings.Join(parts[:len(parts)-1], " ")
			} else {
				cityQuery = query
			}
		} else {
			cityQuery = query
		}
	}

	// Search through cached stations
	c.stationsMu.RLock()
	defer c.stationsMu.RUnlock()

	var results []models.Port
	for _, station := range c.allStations {
		// Check if city matches
		cityMatch := strings.Contains(strings.ToLower(station.Name), cityQuery) ||
			strings.Contains(strings.ToLower(station.City), cityQuery)

		// If state was specified, also check state
		if stateQuery != "" {
			if cityMatch && station.State == stateQuery {
				results = append(results, station)
			}
		} else {
			if cityMatch {
				results = append(results, station)
			}
		}

		// Limit results to avoid overwhelming the user
		if len(results) >= 20 {
			break
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
			StationID:   s.ID,
			Name:        s.Name,
			City:        city,
			State:       s.State,
			Latitude:    s.Latitude,
			Longitude:   s.Longitude,
			TideStationID: s.ID,
			Type:        "coastal",
		})
	}

	return ports, nil
}

// findNearbyMarineStations finds stations near a given location by searching broader areas
func (c *NOAAStationClient) findNearbyMarineStations(ctx context.Context, query string) []models.Port {
	// Extract state if provided
	var stateQuery string
	if idx := strings.Index(query, ","); idx > 0 {
		stateQuery = strings.TrimSpace(query[idx+1:])
		stateQuery = strings.ToUpper(stateQuery)
	}

	// If state was provided, search for all stations in that state
	// This will show nearby coastal stations
	if stateQuery != "" && len(stateQuery) == 2 {
		stations, err := c.searchByState(ctx, stateQuery)
		if err == nil && len(stations) > 0 {
			// Limit to first 5 stations
			if len(stations) > 5 {
				stations = stations[:5]
			}
			return stations
		}
	}

	return nil
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
