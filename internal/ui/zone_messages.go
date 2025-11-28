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

// Provisioning messages

type provisionStatusMsg string

type provisionResultMsg struct {
	err error
}

// waitForProvisioning returns a message wrapping the channels so the Update loop can subscribe to them
type provisioningStartedMsg struct {
	progressChan <-chan string
	resultChan   <-chan error
}

// Actual command to start and return the channels
func initiateProvisioning() tea.Cmd {
	return func() tea.Msg {
		progressChan := make(chan string)
		resultChan := make(chan error)

		go func() {
			// Small delay to ensure UI is ready
			time.Sleep(100 * time.Millisecond)

			// Provision marine zones
			err := zonelookup.ProvisionDatabaseWithProgress(database.DBPath(), progressChan)
			if err != nil {
				resultChan <- err
				close(progressChan)
				return
			}

			// Provision zipcodes
			err = geocoding.ProvisionZipcodeDatabaseWithProgress(database.DBPath(), progressChan)
			
			resultChan <- err
			close(progressChan) // Signal end of progress
		}()

		return provisioningStartedMsg{
			progressChan: progressChan,
			resultChan:   resultChan,
		}
	}
}

func waitForProvisionStatus(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil // Channel closed
		}
		return provisionStatusMsg(msg)
	}
}

func waitForProvisionResult(ch <-chan error) tea.Cmd {
	return func() tea.Msg {
		err := <-ch
		return provisionResultMsg{err: err}
	}
}
