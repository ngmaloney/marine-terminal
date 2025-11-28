package ports

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ngmaloney/marine-terminal/internal/database"
	"github.com/ngmaloney/marine-terminal/internal/geocoding"
	"github.com/ngmaloney/marine-terminal/internal/models"
	"github.com/ngmaloney/marine-terminal/internal/stations"
)

// Service orchestrates port operations
type Service struct {
	repo     *Repository
	geocoder *geocoding.Geocoder
}

// NewService creates a new port service
func NewService() *Service {
	return &Service{
		repo:     NewRepository(),
		geocoder: geocoding.NewGeocoder(),
	}
}

// CreatePort builds and saves a port configuration
func (s *Service) CreatePort(ctx context.Context, name, inputLocation, marineZoneCode string) (*models.Port, error) {
	// 1. Geocode the location to get Lat/Lon
	loc, err := s.geocoder.Geocode(ctx, inputLocation)
	if err != nil {
		return nil, fmt.Errorf("geocoding location: %w", err)
	}
	if loc == nil {
		return nil, fmt.Errorf("location not found: %s", inputLocation)
	}

	// 2. Find the nearest tide station
	tideStations, err := stations.FindNearbyStations(database.DBPath(), loc.Latitude, loc.Longitude, 50.0) // 50 miles search radius
	if err != nil {
		return nil, fmt.Errorf("finding tide stations: %w", err)
	}
	if len(tideStations) == 0 {
		return nil, fmt.Errorf("no tide stations found near %s", inputLocation)
	}
	nearestTideStation := tideStations[0]

	// 3. Construct the Port object
	port := &models.Port{
		Name:          name,
		MarineZoneID:  marineZoneCode,
		TideStationID: nearestTideStation.ID,
		StationID:     nearestTideStation.ID,
		Latitude:      loc.Latitude,
		Longitude:     loc.Longitude,
	}

	// 4. Parse inputLocation to populate State, City, Zipcode
	populateLocationFields(port, inputLocation)

	// 5. Save to database
	if err := s.repo.SavePort(port); err != nil {
		return nil, fmt.Errorf("saving port: %w", err)
	}

	return port, nil
}

func (s *Service) ListPorts() ([]models.Port, error) {
	return s.repo.ListPorts()
}

func (s *Service) DeletePort(name string) error {
	return s.repo.DeletePort(name)
}

// populateLocationFields parses the input string to set City, State, or Zipcode
func populateLocationFields(port *models.Port, input string) {
	input = strings.TrimSpace(input)
	
	// Check if it's a zipcode (5 digits, optionally hyphen + 4 digits)
	zipRegex := regexp.MustCompile(`^\d{5}(-\d{4})?$`)
	
	if zipRegex.MatchString(input) {
		port.Zipcode = input
		return
	}

	// Assuming "City, State" format
	if idx := strings.Index(input, ","); idx > 0 {
		port.City = strings.TrimSpace(input[:idx])
		if idx+1 < len(input) {
			port.State = strings.TrimSpace(input[idx+1:])
		}
	} else {
		// Just a city? or just a state?
		// We treat it as City for now if not numeric
		port.City = input
	}
}