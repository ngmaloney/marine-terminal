package noaa

import (
	"context"
	"time"

	"github.com/ngmaloney/marine-terminal/internal/models"
)

// WeatherClient defines the interface for fetching weather data from NOAA
type WeatherClient interface {
	// GetMarineConditions retrieves current marine conditions for a location
	GetMarineConditions(ctx context.Context, lat, lon float64) (*models.MarineConditions, error)

	// GetMarineForecast retrieves the 3-day marine forecast (old method - by lat/lon)
	GetMarineForecast(ctx context.Context, lat, lon float64) (*models.ThreeDayForecast, error)

	// GetMarineForecastByZone retrieves marine forecast for a specific zone
	GetMarineForecastByZone(ctx context.Context, marineZone string) (*models.MarineConditions, *models.ThreeDayForecast, error)
}

// TideClient defines the interface for fetching tide data from NOAA CO-OPS
type TideClient interface {
	// GetTidePredictions retrieves tide predictions for the next 3 days
	GetTidePredictions(ctx context.Context, stationID string, startDate, endDate time.Time) (*models.TideData, error)
}

// AlertClient defines the interface for fetching NOAA alerts
type AlertClient interface {
	// GetActiveAlerts retrieves active marine alerts for a location
	GetActiveAlerts(ctx context.Context, lat, lon float64) (*models.AlertData, error)

	// GetActiveAlertsByZone retrieves active alerts for a specific marine zone
	GetActiveAlertsByZone(ctx context.Context, marineZone string) (*models.AlertData, error)
}

// PortClient defines the interface for searching ports/stations
type PortClient interface {
	// SearchByLocation searches for ports by city/state or postal code
	SearchByLocation(ctx context.Context, query string) ([]models.Port, error)

	// GetPortByID retrieves a specific port by its station ID
	GetPortByID(ctx context.Context, stationID string) (*models.Port, error)
}
