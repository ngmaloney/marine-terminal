package models

import "time"

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	SeverityExtreme  AlertSeverity = "Extreme"
	SeveritySevere   AlertSeverity = "Severe"
	SeverityModerate AlertSeverity = "Moderate"
	SeverityMinor    AlertSeverity = "Minor"
	SeverityUnknown  AlertSeverity = "Unknown"
)

// Alert represents a NOAA weather or marine alert
type Alert struct {
	ID          string
	Event       string        // e.g., "Small Craft Advisory", "Gale Warning"
	Headline    string
	Description string
	Severity    AlertSeverity
	Urgency     string        // e.g., "Immediate", "Expected"
	Certainty   string        // e.g., "Likely", "Possible"
	Onset       time.Time
	Expires     time.Time
	Areas       []string      // Affected areas
	Instruction string        // What to do
}

// AlertData contains all active alerts for a location
type AlertData struct {
	Alerts    []Alert
	UpdatedAt time.Time
}

// IsActive checks if an alert is currently active
func (a *Alert) IsActive() bool {
	now := time.Now()
	return now.After(a.Onset) && now.Before(a.Expires)
}

// IsMarine returns true if the alert is marine-related
func (a *Alert) IsMarine() bool {
	marineEvents := map[string]bool{
		"Small Craft Advisory": true,
		"Gale Warning": true,
		"Storm Warning": true,
		"Hurricane Force Wind Warning": true,
		"Special Marine Warning": true,
		"Marine Weather Statement": true,
		"Hazardous Seas Warning": true,
	}
	return marineEvents[a.Event]
}
