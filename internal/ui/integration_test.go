package ui

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ngmaloney/mariner-tui/internal/geocoding"
	"github.com/ngmaloney/mariner-tui/internal/models"
	"github.com/ngmaloney/mariner-tui/internal/zonelookup"
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

// TestIntegration_SearchAndGeocode tests the geocoding workflow
func TestIntegration_SearchAndGeocode(t *testing.T) {
	// Create model
	m := NewModel()

	// Step 1: User types "02633" (Chatham zipcode)
	for _, char := range "02633" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	if m.searchInput.Value() != "02633" {
		t.Errorf("searchInput.Value() = %s, want '02633'", m.searchInput.Value())
	}

	// Step 2: User presses Enter to search
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(enterMsg)
	m = updatedModel.(Model)

	// Verify state transition to loading
	if m.state != StateLoading {
		t.Errorf("state = %v, want StateLoading", m.state)
	}

	// Verify command was returned
	if cmd == nil {
		t.Fatal("Expected command to geocode location")
	}

	// Step 3: Simulate geocoding completing
	mockLocation := &geocoding.Location{
		Latitude:  41.6885,
		Longitude: -69.9511,
		Name:      "Chatham, MA 02633",
	}
	geocodeMsg := geocodeMsg{location: mockLocation}
	updatedModel, cmd = m.Update(geocodeMsg)
	m = updatedModel.(Model)

	// Verify location was set
	if m.location == nil {
		t.Fatal("Expected location to be set after geocoding")
	}

	if m.location.Name != "Chatham, MA 02633" {
		t.Errorf("location.Name = %s, want 'Chatham, MA 02633'", m.location.Name)
	}

	// Verify command to find zones was returned
	if cmd == nil {
		t.Fatal("Expected command to find nearby zones")
	}
}

// TestIntegration_ZoneSelection tests zone list and selection
func TestIntegration_ZoneSelection(t *testing.T) {
	// Set up mock data
	mockConditions := &models.MarineConditions{
		Location:    "Cape Cod Bay",
		Temperature: 58,
		Conditions:  "Partly Cloudy",
		UpdatedAt:   time.Now(),
	}

	mockForecast := &models.ThreeDayForecast{
		Periods:   []models.MarineForecast{{PeriodName: "Tonight", Conditions: "Clear"}},
		UpdatedAt: time.Now(),
	}

	mockAlerts := &models.AlertData{
		Alerts:    []models.Alert{},
		UpdatedAt: time.Now(),
	}

	// Create model with mock clients
	m := NewModel()
	m.weatherClient = &mockWeatherClient{conditions: mockConditions, forecast: mockForecast}
	m.alertClient = &mockAlertClient{alerts: mockAlerts}

	// Set up as if we've found zones
	m.state = StateZoneList
	m.zones = []zonelookup.ZoneInfo{
		{Code: "ANZ251", Name: "Cape Cod Bay", Distance: 5.2},
		{Code: "ANZ250", Name: "Coastal Waters East of Cape Cod", Distance: 12.8},
	}
	m.zoneList = createZoneList(m.zones, 80, 20)

	// Step 1: User presses Enter to select first zone
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(enterMsg)
	m = updatedModel.(Model)

	// Verify zone was selected
	if m.selectedZone == nil {
		t.Fatal("Expected zone to be selected")
	}

	if m.selectedZone.Code != "ANZ251" {
		t.Errorf("selectedZone.Code = %s, want 'ANZ251'", m.selectedZone.Code)
	}

	// Verify state transition to loading
	if m.state != StateLoading {
		t.Errorf("state = %v, want StateLoading", m.state)
	}

	// Verify loading states
	if !m.loadingWeather {
		t.Error("loadingWeather should be true after zone selection")
	}
	if !m.loadingAlerts {
		t.Error("loadingAlerts should be true after zone selection")
	}

	// Verify command was returned
	if cmd == nil {
		t.Fatal("Expected command to fetch zone data")
	}

	// Step 2: Simulate weather data arriving
	weatherMsg := zoneWeatherFetchedMsg{
		conditions: mockConditions,
		forecast:   mockForecast,
	}
	updatedModel, _ = m.Update(weatherMsg)
	m = updatedModel.(Model)

	if m.weather == nil {
		t.Error("Weather data should be set after zoneWeatherFetchedMsg")
	}
	if m.forecast == nil {
		t.Error("Forecast data should be set after zoneWeatherFetchedMsg")
	}
	if m.loadingWeather {
		t.Error("loadingWeather should be false after data received")
	}

	// Step 3: Simulate alerts arriving
	alertsMsg := zoneAlertsFetchedMsg{alerts: mockAlerts}
	updatedModel, _ = m.Update(alertsMsg)
	m = updatedModel.(Model)

	if m.alerts == nil {
		t.Error("Alert data should be set after zoneAlertsFetchedMsg")
	}
	if m.loadingAlerts {
		t.Error("loadingAlerts should be false after data received")
	}

	// Step 4: Verify state transition to display
	if m.state != StateDisplay {
		t.Errorf("state = %v, want StateDisplay", m.state)
	}

	// Verify all data is present
	if m.weather.Temperature != 58 {
		t.Errorf("Weather temperature = %.0f, want 58", m.weather.Temperature)
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
	m.alertClient = &mockAlertClient{alerts: &models.AlertData{}}

	// Set up as if we've selected a zone
	m.state = StateLoading
	m.selectedZone = &zonelookup.ZoneInfo{Code: "ANZ251", Name: "Cape Cod Bay"}
	m.loadingWeather = true
	m.loadingAlerts = true

	// Simulate weather fetch failing
	weatherMsg := zoneWeatherFetchedMsg{err: testErr}
	updatedModel, _ := m.Update(weatherMsg)
	m = updatedModel.(Model)

	// Should gracefully handle error - clear loading state but keep UI stable
	if m.loadingWeather {
		t.Error("loadingWeather should be false after error")
	}

	if m.weather != nil {
		t.Error("weather should remain nil after error")
	}

	// Alerts should still work
	alertsMsg := zoneAlertsFetchedMsg{alerts: &models.AlertData{}}
	updatedModel, _ = m.Update(alertsMsg)
	m = updatedModel.(Model)

	if m.alerts == nil {
		t.Error("Alerts should be set even if weather failed")
	}

	// State should transition to Display once both are done (graceful degradation)
	if m.state != StateDisplay {
		t.Error("Should transition to Display state when all loading is done")
	}
}

// TestIntegration_SearchAgain tests returning to search from display
func TestIntegration_SearchAgain(t *testing.T) {
	m := NewModel()

	// Set up as if we're in display mode
	m.state = StateDisplay
	m.selectedZone = &zonelookup.ZoneInfo{Code: "ANZ251", Name: "Cape Cod Bay"}
	m.weather = &models.MarineConditions{Location: "Cape Cod Bay"}

	// User presses 'S' to search again
	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	updatedModel, _ := m.Update(sMsg)
	m = updatedModel.(Model)

	// Verify state transition to search
	if m.state != StateSearch {
		t.Errorf("state = %v, want StateSearch", m.state)
	}

	// Verify data was cleared
	if m.selectedZone != nil {
		t.Error("selectedZone should be cleared when returning to search")
	}
	if m.weather != nil {
		t.Error("weather should be cleared when returning to search")
	}
	if m.forecast != nil {
		t.Error("forecast should be cleared when returning to search")
	}
	if m.alerts != nil {
		t.Error("alerts should be cleared when returning to search")
	}
}
