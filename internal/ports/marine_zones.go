package ports

import (
	"context"
	"strings"

	"github.com/ngmaloney/mariner-tui/internal/models"
	"github.com/ngmaloney/mariner-tui/internal/noaa"
)

// marineZonesByStation maps known station IDs to their marine forecast zones
var marineZonesByStation = map[string]string{
	// Massachusetts
	"8447435": "ANZ254", // Chatham Harbor, Aunt Lydias Cove - Nantucket Sound
	"8447505": "ANZ254", // Chatham, Stage Harbor
	"8447930": "ANZ254", // Woods Hole
	"8449130": "ANZ230", // Nantucket Island
	"8443970": "ANZ237", // Boston

	// Washington
	"9447130": "PZZ131", // Seattle

	// California
	"9414290": "PZZ530", // San Francisco
	"9410170": "PZZ530", // San Diego

	// New York
	"8518750": "ANZ330", // The Battery - New York Harbor
}

// marineZonesByArea maps general areas to marine zones (fallback)
var marineZonesByArea = map[string]string{
	"chatham":   "ANZ254",
	"nantucket": "ANZ230",
	"boston":    "ANZ237",
	"seattle":   "PZZ131",
}

// PopulateMarineZones adds marine zone information to ports
func PopulateMarineZones(ctx context.Context, ports []models.Port) []models.Port {
	result := make([]models.Port, len(ports))

	for i, port := range ports {
		zone := getMarineZone(ctx, &port)
		port.MarineZone = zone
		result[i] = port
	}

	return result
}

// getMarineZone determines the marine forecast zone for a port
func getMarineZone(ctx context.Context, port *models.Port) string {
	// Try exact station ID match first
	if zone, ok := marineZonesByStation[port.ID]; ok {
		return zone
	}

	// Try area-based match
	cityLower := strings.ToLower(port.City)
	for area, zone := range marineZonesByArea {
		if strings.Contains(cityLower, area) {
			return zone
		}
	}

	// Fall back to API lookup
	zone, err := noaa.GetMarineZone(ctx, port.Latitude, port.Longitude)
	if err == nil {
		return zone
	}

	// Return empty if we can't determine the zone
	return ""
}
