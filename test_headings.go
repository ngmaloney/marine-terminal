// Test to verify column headings are visible
package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ngmaloney/mariner-tui/internal/models"
	"github.com/ngmaloney/mariner-tui/internal/ui"
)

func main() {
	// Create a model in display state
	m := ui.NewModel()
	m.SetState(ui.StateDisplay)

	// Set a test port
	port := &models.Port{
		ID:          "8447435",
		Name:        "Chatham Harbor",
		State:       "MA",
		TideStation: "8447435",
		MarineZone:  "ANZ254",
	}
	m.SetCurrentPort(port)

	// Set some test data
	weather := &models.MarineConditions{
		Location: "ANZ254",
		Wind: models.WindData{
			Direction: "W",
			SpeedMin:  15,
			SpeedMax:  20,
		},
		Seas: models.SeaState{
			HeightMin: 5,
			HeightMax: 7,
		},
	}
	m.SetWeather(weather)

	forecast := &models.ThreeDayForecast{
		Periods: []models.MarineForecast{
			{PeriodName: "THIS AFTERNOON", Conditions: "Windy"},
			{PeriodName: "TONIGHT", Conditions: "Clear"},
		},
	}
	m.SetForecast(forecast)

	tides := &models.TideData{
		StationID: "8447435",
		Events: []models.TideEvent{
			{Type: models.TideHigh, Height: 5.0},
		},
	}
	m.SetTides(tides)

	alerts := &models.AlertData{
		Alerts: []models.Alert{},
	}
	m.SetAlerts(alerts)

	// Simulate window size
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 150, Height: 40})
	m = updatedModel.(ui.Model)

	// Render and print the view
	view := m.View()

	fmt.Println("=== RENDERED VIEW ===")
	fmt.Println(view)
	fmt.Println("\n=== CHECKING FOR HEADINGS ===")

	// Check if headings are present
	headings := []string{
		"WEATHER",
		"TIDES",
		"ALERTS",
	}

	for _, heading := range headings {
		if containsString(view, heading) {
			fmt.Printf("✓ Found: %s\n", heading)
		} else {
			fmt.Printf("✗ Missing: %s\n", heading)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr ||
		containsStringHelper(s[1:], substr)))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
