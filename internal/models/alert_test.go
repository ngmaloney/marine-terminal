package models

import (
	"testing"
	"time"
)

func TestAlert_IsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		alert  Alert
		want   bool
	}{
		{
			name: "currently active alert",
			alert: Alert{
				Onset:   now.Add(-1 * time.Hour),
				Expires: now.Add(2 * time.Hour),
			},
			want: true,
		},
		{
			name: "expired alert",
			alert: Alert{
				Onset:   now.Add(-3 * time.Hour),
				Expires: now.Add(-1 * time.Hour),
			},
			want: false,
		},
		{
			name: "future alert",
			alert: Alert{
				Onset:   now.Add(1 * time.Hour),
				Expires: now.Add(3 * time.Hour),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.alert.IsActive()
			if got != tt.want {
				t.Errorf("Alert.IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlert_IsMarine(t *testing.T) {
	tests := []struct {
		name  string
		event string
		want  bool
	}{
		{"small craft advisory", "Small Craft Advisory", true},
		{"gale warning", "Gale Warning", true},
		{"storm warning", "Storm Warning", true},
		{"hurricane force wind", "Hurricane Force Wind Warning", true},
		{"special marine warning", "Special Marine Warning", true},
		{"marine weather statement", "Marine Weather Statement", true},
		{"hazardous seas", "Hazardous Seas Warning", true},
		{"not marine - tornado", "Tornado Warning", false},
		{"not marine - flood", "Flood Warning", false},
		{"not marine - winter storm", "Winter Storm Warning", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := Alert{Event: tt.event}
			got := alert.IsMarine()
			if got != tt.want {
				t.Errorf("Alert.IsMarine() for %q = %v, want %v", tt.event, got, tt.want)
			}
		})
	}
}

func TestAlertSeverity_Constants(t *testing.T) {
	tests := []struct {
		severity AlertSeverity
		want     string
	}{
		{SeverityExtreme, "Extreme"},
		{SeveritySevere, "Severe"},
		{SeverityModerate, "Moderate"},
		{SeverityMinor, "Minor"},
		{SeverityUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			if string(tt.severity) != tt.want {
				t.Errorf("Severity constant = %v, want %v", tt.severity, tt.want)
			}
		})
	}
}
