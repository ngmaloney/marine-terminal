package models

import (
	"testing"
	"time"
)

func TestTideData_GetEventsForDay(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	tests := []struct {
		name   string
		events []TideEvent
		date   time.Time
		want   int // number of events expected
	}{
		{
			name: "typical day with 2 highs and 2 lows",
			events: []TideEvent{
				{Time: time.Date(2025, 11, 27, 6, 30, 0, 0, loc), Type: TideLow, Height: 0.5},
				{Time: time.Date(2025, 11, 27, 12, 45, 0, 0, loc), Type: TideHigh, Height: 5.2},
				{Time: time.Date(2025, 11, 27, 18, 15, 0, 0, loc), Type: TideLow, Height: 0.8},
				{Time: time.Date(2025, 11, 28, 0, 30, 0, 0, loc), Type: TideHigh, Height: 5.0},
			},
			date: time.Date(2025, 11, 27, 0, 0, 0, 0, loc),
			want: 3,
		},
		{
			name: "no events for given day",
			events: []TideEvent{
				{Time: time.Date(2025, 11, 26, 12, 0, 0, 0, loc), Type: TideHigh, Height: 5.0},
				{Time: time.Date(2025, 11, 28, 12, 0, 0, 0, loc), Type: TideHigh, Height: 5.0},
			},
			date: time.Date(2025, 11, 27, 0, 0, 0, 0, loc),
			want: 0,
		},
		{
			name: "single event",
			events: []TideEvent{
				{Time: time.Date(2025, 11, 27, 12, 0, 0, 0, loc), Type: TideHigh, Height: 5.0},
			},
			date: time.Date(2025, 11, 27, 0, 0, 0, 0, loc),
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := &TideData{
				StationID: "TEST123",
				Events:    tt.events,
			}
			got := td.GetEventsForDay(tt.date)
			if len(got) != tt.want {
				t.Errorf("GetEventsForDay() returned %d events, want %d", len(got), tt.want)
			}
		})
	}
}

func TestTideType_Constants(t *testing.T) {
	if TideHigh != "H" {
		t.Errorf("TideHigh = %v, want 'H'", TideHigh)
	}
	if TideLow != "L" {
		t.Errorf("TideLow = %v, want 'L'", TideLow)
	}
}
