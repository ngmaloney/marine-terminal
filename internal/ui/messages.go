package ui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/ngmaloney/marine-terminal/internal/database"
	"github.com/ngmaloney/marine-terminal/internal/models"
	"github.com/ngmaloney/marine-terminal/internal/zonelookup"
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

// directLoadStationMsg is sent when a station is directly loaded by code
type directLoadStationMsg struct {
	zone *zonelookup.ZoneInfo
	err  error
}

// directLoadStation attempts to load zone info for a given station code
func directLoadStation(zoneCode string) tea.Cmd {
	return func() tea.Msg {
		zone, err := zonelookup.GetZoneInfoByCode(database.DBPath(), zoneCode)
		return directLoadStationMsg{zone: zone, err: err}
	}
}
