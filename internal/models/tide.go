package models

import "time"

// TideType represents whether a tide is high or low
type TideType string

const (
	TideHigh TideType = "H"
	TideLow  TideType = "L"
)

// TideEvent represents a single high or low tide occurrence
type TideEvent struct {
	Time   time.Time
	Type   TideType
	Height float64 // feet relative to MLLW (Mean Lower Low Water)
}

// TideData contains tide predictions for a location
type TideData struct {
	StationID   string
	StationName string
	Events      []TideEvent // Ordered by time
	UpdatedAt   time.Time
}

// GetEventsForDay returns tide events for a specific date
func (td *TideData) GetEventsForDay(date time.Time) []TideEvent {
	var events []TideEvent
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	for _, event := range td.Events {
		if event.Time.After(startOfDay) && event.Time.Before(endOfDay) {
			events = append(events, event)
		}
	}
	return events
}
