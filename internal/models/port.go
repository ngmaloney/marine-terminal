package models

// Port represents a marine location/station
type Port struct {
	ID          string  // NOAA station ID
	Name        string
	State       string
	City        string
	Latitude    float64
	Longitude   float64
	TideStation string // May differ from main station ID for tide predictions
	MarineZone  string // NOAA marine forecast zone (e.g., "ANZ254")
	Type        string // e.g., "buoy", "coastal", "offshore"
}
