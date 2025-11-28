package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ngmaloney/marine-terminal/internal/database"
	"github.com/ngmaloney/marine-terminal/internal/geocoding"
	"github.com/ngmaloney/marine-terminal/internal/models"
	"github.com/ngmaloney/marine-terminal/internal/noaa"
	"github.com/ngmaloney/marine-terminal/internal/zonelookup"
)

// geocodeMsg is sent when geocoding completes
type geocodeMsg struct {
	location *geocoding.Location
	err      error
}

// zonesFoundMsg is sent when nearby zones are found
type zonesFoundMsg struct {
	zones []zonelookup.ZoneInfo
	err   error
}

// zoneWeatherFetchedMsg is sent when weather data for a zone is fetched
type zoneWeatherFetchedMsg struct {
	conditions *models.MarineConditions
	forecast   *models.ThreeDayForecast
	err        error
}

// zoneAlertsFetchedMsg is sent when alerts for a zone are fetched
type zoneAlertsFetchedMsg struct {
	alerts *models.AlertData
	err    error
}

// geocodeLocation performs geocoding in the background
func geocodeLocation(geocoder *geocoding.Geocoder, query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		location, err := geocoder.Geocode(ctx, query)
		return geocodeMsg{location: location, err: err}
	}
}

// findNearbyZones finds marine zones near a location
func findNearbyZones(lat, lon float64) tea.Cmd {
	return func() tea.Msg {
		zones, err := zonelookup.GetNearbyMarineZones(database.DBPath(), lat, lon, 50.0)
		return zonesFoundMsg{zones: zones, err: err}
	}
}

// fetchZoneWeather fetches weather data for a marine zone
func fetchZoneWeather(client noaa.WeatherClient, zoneCode string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		conditions, forecast, err := client.GetMarineForecastByZone(ctx, zoneCode)
		return zoneWeatherFetchedMsg{
			conditions: conditions,
			forecast:   forecast,
			err:        err,
		}
	}
}

// fetchZoneAlerts fetches alerts for a marine zone
func fetchZoneAlerts(client noaa.AlertClient, zoneCode string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		alerts, err := client.GetActiveAlertsByZone(ctx, zoneCode)
		return zoneAlertsFetchedMsg{alerts: alerts, err: err}
	}
}
