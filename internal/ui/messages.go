package ui

import (

	"github.com/ngmaloney/mariner-tui/internal/models"
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
