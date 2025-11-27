package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ngmaloney/mariner-tui/internal/models"
)

// Mock clients for testing

type mockWeatherClient struct {
	conditions *models.MarineConditions
	forecast   *models.ThreeDayForecast
	err        error
}

func (m *mockWeatherClient) GetMarineConditions(ctx context.Context, lat, lon float64) (*models.MarineConditions, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.conditions, nil
}

func (m *mockWeatherClient) GetMarineForecast(ctx context.Context, lat, lon float64) (*models.ThreeDayForecast, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.forecast, nil
}

func (m *mockWeatherClient) GetMarineForecastByZone(ctx context.Context, marineZone string) (*models.MarineConditions, *models.ThreeDayForecast, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	return m.conditions, m.forecast, nil
}

type mockTideClient struct {
	tides *models.TideData
	err   error
}

func (m *mockTideClient) GetTidePredictions(ctx context.Context, stationID string, startDate, endDate time.Time) (*models.TideData, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tides, nil
}

type mockAlertClient struct {
	alerts *models.AlertData
	err    error
}

func (m *mockAlertClient) GetActiveAlerts(ctx context.Context, lat, lon float64) (*models.AlertData, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.alerts, nil
}

func (m *mockAlertClient) GetActiveAlertsByZone(ctx context.Context, marineZone string) (*models.AlertData, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.alerts, nil
}

// TestIntegration_SearchAndFetchData tests the complete workflow
func TestIntegration_SearchAndFetchData(t *testing.T) {
	// Set up mock data
	mockConditions := &models.MarineConditions{
		Location:    "Seattle, WA",
		Temperature: 58,
		Conditions:  "Partly Cloudy",
		UpdatedAt:   time.Now(),
	}

	mockForecast := &models.ThreeDayForecast{
		Periods:   []models.MarineForecast{{PeriodName: "Tonight", Conditions: "Clear"}},
		UpdatedAt: time.Now(),
	}

	mockTides := &models.TideData{
		StationID: "9447130",
		Events:    []models.TideEvent{{Time: time.Now(), Type: models.TideHigh, Height: 5.0}},
		UpdatedAt: time.Now(),
	}

	mockAlerts := &models.AlertData{
		Alerts:    []models.Alert{},
		UpdatedAt: time.Now(),
	}

	// Create model with mock clients
	m := NewModel()
	m.weatherClient = &mockWeatherClient{conditions: mockConditions, forecast: mockForecast}
	m.tideClient = &mockTideClient{tides: mockTides}
	m.alertClient = &mockAlertClient{alerts: mockAlerts}

	// Step 1: User types "Seattle"
	for _, char := range "Seattle" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	if m.searchInput.Value() != "Seattle" {
		t.Errorf("searchInput.Value() = %s, want 'Seattle'", m.searchInput.Value())
	}

	// Step 2: User presses Enter to search
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(enterMsg)
	m = updatedModel.(Model)

	// Verify port was found
	if m.currentPort == nil {
		t.Fatal("Expected port to be selected after search")
	}

	// NOAA API returns full names like "SEATTLE (Madison St.), Elliott Bay"
	if !strings.Contains(strings.ToLower(m.currentPort.Name), "seattle") && m.currentPort.State != "WA" {
		t.Errorf("Selected port = %s, want a Seattle/WA port", m.currentPort.Name)
	}

	// Verify state transition
	if m.state != StateDisplay {
		t.Errorf("state = %v, want StateDisplay", m.state)
	}

	// Verify loading states were set
	if !m.loadingWeather {
		t.Error("loadingWeather should be true after port selection")
	}
	if !m.loadingTides {
		t.Error("loadingTides should be true after port selection")
	}
	if !m.loadingAlerts {
		t.Error("loadingAlerts should be true after port selection")
	}

	// Verify command was returned
	if cmd == nil {
		t.Fatal("Expected command to fetch data")
	}

	// Step 4: Execute the commands (simulates async API calls)
	// In real Bubble Tea, these would run in goroutines and send messages back
	// For testing, we can simulate the messages

	// Simulate weather data arriving
	weatherMsg := weatherFetchedMsg{
		conditions: mockConditions,
		forecast:   mockForecast,
	}
	updatedModel, _ = m.Update(weatherMsg)
	m = updatedModel.(Model)

	if m.weather == nil {
		t.Error("Weather data should be set after weatherFetchedMsg")
	}
	if m.forecast == nil {
		t.Error("Forecast data should be set after weatherFetchedMsg")
	}
	if m.loadingWeather {
		t.Error("loadingWeather should be false after data received")
	}

	// Simulate tide data arriving
	tidesMsg := tidesFetchedMsg{tides: mockTides}
	updatedModel, _ = m.Update(tidesMsg)
	m = updatedModel.(Model)

	if m.tides == nil {
		t.Error("Tide data should be set after tidesFetchedMsg")
	}
	if m.loadingTides {
		t.Error("loadingTides should be false after data received")
	}

	// Simulate alerts arriving
	alertsMsg := alertsFetchedMsg{alerts: mockAlerts}
	updatedModel, _ = m.Update(alertsMsg)
	m = updatedModel.(Model)

	if m.alerts == nil {
		t.Error("Alert data should be set after alertsFetchedMsg")
	}
	if m.loadingAlerts {
		t.Error("loadingAlerts should be false after data received")
	}

	// Verify all data is present
	if m.weather.Temperature != 58 {
		t.Errorf("Weather temperature = %.0f, want 58", m.weather.Temperature)
	}
	if len(m.tides.Events) != 1 {
		t.Errorf("Tide events count = %d, want 1", len(m.tides.Events))
	}
	if len(m.alerts.Alerts) != 0 {
		t.Errorf("Alerts count = %d, want 0", len(m.alerts.Alerts))
	}
}

// TestIntegration_ErrorHandling tests graceful error handling
func TestIntegration_ErrorHandling(t *testing.T) {
	m := NewModel()

	// Set up clients that will fail
	testErr := fmt.Errorf("API timeout")
	m.weatherClient = &mockWeatherClient{err: testErr}
	m.tideClient = &mockTideClient{tides: &models.TideData{StationID: "9447130"}}
	m.alertClient = &mockAlertClient{alerts: &models.AlertData{}}

	// Search for port
	for _, char := range "Seattle" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	// Press Enter
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(enterMsg)
	m = updatedModel.(Model)

	// Simulate weather fetch failing
	weatherMsg := weatherFetchedMsg{err: testErr}
	updatedModel, _ = m.Update(weatherMsg)
	m = updatedModel.(Model)

	// Should gracefully handle error - clear loading state but keep UI stable
	if m.loadingWeather {
		t.Error("loadingWeather should be false after error")
	}

	if m.weather != nil {
		t.Error("weather should remain nil after error")
	}

	// Tides should still work
	tidesMsg := tidesFetchedMsg{tides: &models.TideData{StationID: "9447130"}}
	updatedModel, _ = m.Update(tidesMsg)
	m = updatedModel.(Model)

	if m.tides == nil {
		t.Error("Tides should be set even if weather failed")
	}

	// State should still be Display (graceful degradation)
	if m.state != StateDisplay {
		t.Error("Should stay in Display state despite errors")
	}
}
