package ports

import (
	"context"
	"fmt"
	"strings"

	"github.com/ngmaloney/mariner-tui/internal/models"
)

// Client defines the interface for searching ports
type Client interface {
	SearchByLocation(ctx context.Context, query string) ([]models.Port, error)
	GetPortByID(ctx context.Context, stationID string) (*models.Port, error)
}

// StaticPortClient provides a simple in-memory port database
// In production, this could be replaced with an API-based lookup
type StaticPortClient struct {
	ports []models.Port
}

// NewStaticPortClient creates a client with predefined US ports
func NewStaticPortClient() *StaticPortClient {
	return &StaticPortClient{
		ports: getDefaultPorts(),
	}
}

// SearchByLocation searches for ports by city, state, or postal code
func (c *StaticPortClient) SearchByLocation(ctx context.Context, query string) ([]models.Port, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	var results []models.Port
	for _, port := range c.ports {
		// Search by city, state, or name
		if strings.Contains(strings.ToLower(port.City), query) ||
			strings.Contains(strings.ToLower(port.State), query) ||
			strings.Contains(strings.ToLower(port.Name), query) {
			results = append(results, port)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no ports found for '%s'. Try: city name, state (CA, WA)", query)
	}

	return results, nil
}

// GetPortByID retrieves a specific port by station ID
func (c *StaticPortClient) GetPortByID(ctx context.Context, stationID string) (*models.Port, error) {
	for _, port := range c.ports {
		if port.ID == stationID {
			return &port, nil
		}
	}
	return nil, fmt.Errorf("port not found: %s", stationID)
}

// getDefaultPorts returns a list of major US marine ports
func getDefaultPorts() []models.Port {
	return []models.Port{
		// Massachusetts ports
		{
			ID:          "8447930",
			Name:        "Woods Hole",
			City:        "Woods Hole",
			State:       "MA",
			Latitude:    41.5233,
			Longitude:   -70.6717,
			TideStation: "8447930",
			Type:        "coastal",
		},
		{
			ID:          "8449130",
			Name:        "Nantucket Island",
			City:        "Nantucket",
			State:       "MA",
			Latitude:    41.2833,
			Longitude:   -70.0967,
			TideStation: "8449130",
			Type:        "coastal",
		},
		{
			ID:          "8447386",
			Name:        "Chatham, Aunt Lydias Cove",
			City:        "Chatham",
			State:       "MA",
			Latitude:    41.6800,
			Longitude:   -69.9567,
			TideStation: "8447386",
			Type:        "coastal",
		},
		{
			ID:          "8443970",
			Name:        "Boston",
			City:        "Boston",
			State:       "MA",
			Latitude:    42.3601,
			Longitude:   -71.0589,
			TideStation: "8443970",
			Type:        "coastal",
		},
		{
			ID:          "8447270",
			Name:        "Chatham Harbor",
			City:        "Chatham",
			State:       "MA",
			Latitude:    41.6667,
			Longitude:   -69.9500,
			TideStation: "8447270",
			Type:        "coastal",
		},
		{
			ID:          "9447130",
			Name:        "Seattle",
			City:        "Seattle",
			State:       "WA",
			Latitude:    47.6062,
			Longitude:   -122.3321,
			TideStation: "9447130",
			Type:        "coastal",
		},
		{
			ID:          "9414290",
			Name:        "San Francisco",
			City:        "San Francisco",
			State:       "CA",
			Latitude:    37.8063,
			Longitude:   -122.4659,
			TideStation: "9414290",
			Type:        "coastal",
		},
		{
			ID:          "8518750",
			Name:        "The Battery",
			City:        "New York",
			State:       "NY",
			Latitude:    40.7006,
			Longitude:   -74.0142,
			TideStation: "8518750",
			Type:        "coastal",
		},
		{
			ID:          "8729108",
			Name:        "Panama City",
			City:        "Panama City",
			State:       "FL",
			Latitude:    30.1517,
			Longitude:   -85.6667,
			TideStation: "8729108",
			Type:        "coastal",
		},
		{
			ID:          "8771450",
			Name:        "Galveston Pier 21",
			City:        "Galveston",
			State:       "TX",
			Latitude:    29.3102,
			Longitude:   -94.7933,
			TideStation: "8771450",
			Type:        "coastal",
		},
		{
			ID:          "9410170",
			Name:        "San Diego",
			City:        "San Diego",
			State:       "CA",
			Latitude:    32.7141,
			Longitude:   -117.1731,
			TideStation: "9410170",
			Type:        "coastal",
		},
		{
			ID:          "8454000",
			Name:        "Providence",
			City:        "Providence",
			State:       "RI",
			Latitude:    41.8070,
			Longitude:   -71.4012,
			TideStation: "8454000",
			Type:        "coastal",
		},
		{
			ID:          "8467150",
			Name:        "Bridgeport",
			City:        "Bridgeport",
			State:       "CT",
			Latitude:    41.1739,
			Longitude:   -73.1817,
			TideStation: "8467150",
			Type:        "coastal",
		},
		{
			ID:          "9435380",
			Name:        "Astoria",
			City:        "Astoria",
			State:       "OR",
			Latitude:    46.2071,
			Longitude:   -123.7689,
			TideStation: "9435380",
			Type:        "coastal",
		},
		{
			ID:          "8658120",
			Name:        "Wilmington",
			City:        "Wilmington",
			State:       "NC",
			Latitude:    34.2272,
			Longitude:   -77.9531,
			TideStation: "8658120",
			Type:        "coastal",
		},
	}
}
