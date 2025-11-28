package models

import "time"

// Port represents a user-configured marine location.
// It can be a transient object from an API search or a saved user configuration.
type Port struct {
	ID            int64     `json:"id"`             // Database Primary Key (0 if not saved)
	StationID     string    `json:"station_id"`     // NOAA station ID (e.g. "8447435")
	Name          string    `json:"name"`           // User-friendly name
	State         string    `json:"state"`          // State (e.g. "MA")
	City          string    `json:"city"`           // City (e.g. "Chatham")
	Zipcode       string    `json:"zipcode"`        // Zipcode (e.g. "02633")
	MarineZoneID  string    `json:"marine_zone_id"` // NOAA marine forecast zone (e.g. "ANZ254")
	TideStationID string    `json:"tide_station_id"`// NOAA tide station ID
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Type          string    `json:"type"`           // e.g., "buoy", "coastal"
	CreatedAt     time.Time `json:"created_at"`
}