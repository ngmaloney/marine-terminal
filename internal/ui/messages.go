package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ngmaloney/mariner-tui/internal/models"
	"github.com/ngmaloney/mariner-tui/internal/noaa"
)

// Message types for async operations

// weatherFetchedMsg is sent when weather data has been fetched
type weatherFetchedMsg struct {
	conditions *models.MarineConditions
	forecast   *models.ThreeDayForecast
	err        error
}

// tidesFetchedMsg is sent when tide data has been fetched
type tidesFetchedMsg struct {
	tides *models.TideData
	err   error
}

// alertsFetchedMsg is sent when alert data has been fetched
type alertsFetchedMsg struct {
	alerts *models.AlertData
	err    error
}

// errMsg is a message type for errors
type errMsg struct {
	err error
}

// Commands for fetching data

// fetchWeatherData fetches weather and forecast data for a port
func fetchWeatherData(port *models.Port, client noaa.WeatherClient) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// If we have a marine zone, use marine forecast by zone
		if port.MarineZone != "" {
			conditions, forecast, err := client.GetMarineForecastByZone(ctx, port.MarineZone)
			if err != nil {
				return weatherFetchedMsg{err: err}
			}
			return weatherFetchedMsg{
				conditions: conditions,
				forecast:   forecast,
			}
		}

		// Fall back to lat/lon based forecast
		conditions, err := client.GetMarineConditions(ctx, port.Latitude, port.Longitude)
		if err != nil {
			return weatherFetchedMsg{err: err}
		}

		forecast, err := client.GetMarineForecast(ctx, port.Latitude, port.Longitude)
		if err != nil {
			return weatherFetchedMsg{conditions: conditions, err: err}
		}

		return weatherFetchedMsg{
			conditions: conditions,
			forecast:   forecast,
		}
	}
}

// fetchTideData fetches tide predictions for a port
func fetchTideData(port *models.Port, client noaa.TideClient) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get tides for next 3 days
		startDate := time.Now()
		endDate := startDate.Add(3 * 24 * time.Hour)

		tides, err := client.GetTidePredictions(ctx, port.TideStation, startDate, endDate)
		if err != nil {
			return tidesFetchedMsg{err: err}
		}

		return tidesFetchedMsg{tides: tides}
	}
}

// fetchAlertData fetches active alerts for a port
func fetchAlertData(port *models.Port, client noaa.AlertClient) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// If we have a marine zone, use zone-based alerts
		if port.MarineZone != "" {
			alerts, err := client.GetActiveAlertsByZone(ctx, port.MarineZone)
			if err != nil {
				return alertsFetchedMsg{err: err}
			}
			return alertsFetchedMsg{alerts: alerts}
		}

		// Fall back to lat/lon based alerts
		alerts, err := client.GetActiveAlerts(ctx, port.Latitude, port.Longitude)
		if err != nil {
			return alertsFetchedMsg{err: err}
		}

		return alertsFetchedMsg{alerts: alerts}
	}
}

// fetchAllData returns a batch command to fetch all data for a port
func fetchAllData(port *models.Port, weatherClient noaa.WeatherClient, tideClient noaa.TideClient, alertClient noaa.AlertClient) tea.Cmd {
	return tea.Batch(
		fetchWeatherData(port, weatherClient),
		fetchTideData(port, tideClient),
		fetchAlertData(port, alertClient),
	)
}
