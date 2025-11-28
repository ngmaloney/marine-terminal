package ports

import (
	"context"

	"github.com/ngmaloney/marine-terminal/internal/database"
	"github.com/ngmaloney/marine-terminal/internal/models"
	"github.com/ngmaloney/marine-terminal/internal/zonelookup"
)

// PopulateMarineZones adds marine zone information to ports using SQLite database lookup
// Sets the closest zone as the primary zone
func PopulateMarineZones(ctx context.Context, ports []models.Port) []models.Port {
	result := make([]models.Port, len(ports))

	for i, port := range ports {
		zone := getClosestMarineZone(ctx, &port)
		port.MarineZone = zone
		result[i] = port
	}

	return result
}

// getClosestMarineZone determines the closest marine forecast zone for a port
func getClosestMarineZone(ctx context.Context, port *models.Port) string {
	// Get zones within 50 miles
	zones, err := zonelookup.GetNearbyMarineZones(database.DBPath(), port.Latitude, port.Longitude, 50.0)
	if err != nil {
		return ""
	}
	if len(zones) == 0 {
		return ""
	}

	// Return the closest zone
	return zones[0].Code
}

