package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ngmaloney/mariner-tui/internal/models"
	"github.com/ngmaloney/mariner-tui/internal/ui"
)

// This demo shows the UI with mock data
func main() {
	// Create a model with mock data
	m := ui.NewModel()

	// Set up mock port
	m.SetCurrentPort(&models.Port{
		ID:          "9447130",
		Name:        "Seattle",
		City:        "Seattle",
		State:       "WA",
		Latitude:    47.6062,
		Longitude:   -122.3321,
		TideStation: "9447130",
		Type:        "coastal",
	})

	// Set up mock weather data
	m.SetWeather(&models.MarineConditions{
		Location:    "Seattle, WA",
		Temperature: 58,
		Conditions:  "Partly Cloudy",
		Wind: models.WindData{
			Direction: "W",
			SpeedMin:  15,
			SpeedMax:  20,
			GustSpeed: 30,
			HasGust:   true,
		},
		Seas: models.SeaState{
			HeightMin: 5,
			HeightMax: 7,
			Components: []models.WaveComponent{
				{Direction: "S", Height: 5.0, Period: 8},
				{Direction: "W", Height: 4.0, Period: 5},
			},
		},
		UpdatedAt: time.Now(),
	})

	// Set up mock forecast
	now := time.Now()
	m.SetForecast(&models.ThreeDayForecast{
		Periods: []models.MarineForecast{
			{
				Date:        now,
				DayOfWeek:   now.Weekday().String(),
				PeriodName:  "This Afternoon",
				Conditions:  "Partly Cloudy",
				Temperature: 58,
				Wind: models.WindData{
					Direction: "W",
					SpeedMin:  15,
					SpeedMax:  20,
					GustSpeed: 30,
					HasGust:   true,
				},
			},
			{
				Date:        now.Add(12 * time.Hour),
				DayOfWeek:   now.Weekday().String(),
				PeriodName:  "Tonight",
				Conditions:  "Mostly Clear",
				Temperature: 45,
				Wind: models.WindData{
					Direction: "W",
					SpeedMin:  10,
					SpeedMax:  15,
				},
			},
			{
				Date:        now.Add(24 * time.Hour),
				DayOfWeek:   now.Add(24 * time.Hour).Weekday().String(),
				PeriodName:  "Friday",
				Conditions:  "Sunny",
				Temperature: 62,
			},
		},
		UpdatedAt: time.Now(),
	})

	// Set up mock tide data
	m.SetTides(&models.TideData{
		StationID:   "9447130",
		StationName: "Seattle",
		Events: []models.TideEvent{
			{Time: now.Add(2 * time.Hour), Type: models.TideLow, Height: 0.5},
			{Time: now.Add(8 * time.Hour), Type: models.TideHigh, Height: 5.2},
			{Time: now.Add(14 * time.Hour), Type: models.TideLow, Height: 0.8},
			{Time: now.Add(20 * time.Hour), Type: models.TideHigh, Height: 5.0},
			{Time: now.Add(26 * time.Hour), Type: models.TideLow, Height: 0.6},
			{Time: now.Add(32 * time.Hour), Type: models.TideHigh, Height: 5.3},
		},
		UpdatedAt: time.Now(),
	})

	// Set up mock alerts
	m.SetAlerts(&models.AlertData{
		Alerts: []models.Alert{
			{
				ID:       "alert-1",
				Event:    "Small Craft Advisory",
				Headline: "West winds 15 to 25 kt with gusts up to 35 kt expected",
				Description: "West winds 15 to 25 kt with gusts up to 35 kt. Seas 6 to 9 ft. " +
					"Inexperienced mariners, especially those operating smaller vessels, " +
					"should avoid navigating in hazardous conditions.",
				Severity:    models.SeverityModerate,
				Urgency:     "Expected",
				Certainty:   "Likely",
				Onset:       now.Add(-2 * time.Hour),
				Expires:     now.Add(6 * time.Hour),
				Areas:       []string{"Puget Sound"},
				Instruction: "Small craft should use caution.",
			},
		},
		UpdatedAt: time.Now(),
	})

	// Set state to display
	m.SetState(ui.StateDisplay)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running demo: %v\n", err)
		os.Exit(1)
	}
}
