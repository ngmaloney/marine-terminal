package noaa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ngmaloney/mariner-tui/internal/models"
)

// NOAAWeatherClient implements WeatherClient using the NOAA Weather API
type NOAAWeatherClient struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

// NewWeatherClient creates a new NOAA weather client
func NewWeatherClient() *NOAAWeatherClient {
	return &NOAAWeatherClient{
		baseURL: "https://api.weather.gov",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "MarinerTUI/1.0 (github.com/ngmaloney/mariner-tui)",
	}
}

// GetMarineConditions retrieves current marine conditions
func (c *NOAAWeatherClient) GetMarineConditions(ctx context.Context, lat, lon float64) (*models.MarineConditions, error) {
	// First, get the grid point for this location
	gridPoint, err := c.getGridPoint(ctx, lat, lon)
	if err != nil {
		return nil, fmt.Errorf("failed to get grid point: %w", err)
	}

	// Get the forecast office and grid coordinates
	forecastURL := fmt.Sprintf("%s/gridpoints/%s/%d,%d/forecast",
		c.baseURL, gridPoint.GridID, gridPoint.GridX, gridPoint.GridY)

	req, err := http.NewRequestWithContext(ctx, "GET", forecastURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch forecast: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var forecastResp forecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecastResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our model (simplified - would need more parsing logic)
	conditions := &models.MarineConditions{
		Location:  fmt.Sprintf("%.2f, %.2f", lat, lon),
		UpdatedAt: time.Now(),
	}

	// Parse first period for current conditions
	if len(forecastResp.Properties.Periods) > 0 {
		period := forecastResp.Properties.Periods[0]
		conditions.Conditions = period.ShortForecast
		conditions.Temperature = float64(period.Temperature)
		// Note: Full wind/wave parsing would happen here
	}

	return conditions, nil
}

// GetMarineForecast retrieves the 3-day marine forecast
func (c *NOAAWeatherClient) GetMarineForecast(ctx context.Context, lat, lon float64) (*models.ThreeDayForecast, error) {
	// Similar implementation to GetMarineConditions but returns forecast periods
	gridPoint, err := c.getGridPoint(ctx, lat, lon)
	if err != nil {
		return nil, fmt.Errorf("failed to get grid point: %w", err)
	}

	forecastURL := fmt.Sprintf("%s/gridpoints/%s/%d,%d/forecast",
		c.baseURL, gridPoint.GridID, gridPoint.GridX, gridPoint.GridY)

	req, err := http.NewRequestWithContext(ctx, "GET", forecastURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch forecast: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var forecastResp forecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecastResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert periods to our forecast model
	forecast := &models.ThreeDayForecast{
		UpdatedAt: time.Now(),
		Periods:   make([]models.MarineForecast, 0),
	}

	// Take first 6 periods (3 days with day/night)
	maxPeriods := 6
	if len(forecastResp.Properties.Periods) < maxPeriods {
		maxPeriods = len(forecastResp.Properties.Periods)
	}

	for i := 0; i < maxPeriods; i++ {
		period := forecastResp.Properties.Periods[i]
		startTime, _ := time.Parse(time.RFC3339, period.StartTime)

		marineForecast := models.MarineForecast{
			Date:        startTime,
			DayOfWeek:   startTime.Weekday().String(),
			PeriodName:  period.Name,
			Conditions:  period.ShortForecast,
			Temperature: float64(period.Temperature),
			RawText:     period.DetailedForecast,
		}

		forecast.Periods = append(forecast.Periods, marineForecast)
	}

	return forecast, nil
}

// getGridPoint gets the NOAA grid point for a lat/lon
func (c *NOAAWeatherClient) getGridPoint(ctx context.Context, lat, lon float64) (*gridPoint, error) {
	url := fmt.Sprintf("%s/points/%.4f,%.4f", c.baseURL, lat, lon)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get grid point (status %d): %s", resp.StatusCode, string(body))
	}

	var pointResp pointResponse
	if err := json.NewDecoder(resp.Body).Decode(&pointResp); err != nil {
		return nil, err
	}

	return &gridPoint{
		GridID: pointResp.Properties.GridID,
		GridX:  pointResp.Properties.GridX,
		GridY:  pointResp.Properties.GridY,
	}, nil
}

// Internal types for NOAA API responses

type gridPoint struct {
	GridID string
	GridX  int
	GridY  int
}

type pointResponse struct {
	Properties struct {
		GridID string `json:"gridId"`
		GridX  int    `json:"gridX"`
		GridY  int    `json:"gridY"`
	} `json:"properties"`
}

type forecastResponse struct {
	Properties struct {
		Periods []struct {
			Name             string `json:"name"`
			StartTime        string `json:"startTime"`
			EndTime          string `json:"endTime"`
			Temperature      int    `json:"temperature"`
			TemperatureUnit  string `json:"temperatureUnit"`
			WindSpeed        string `json:"windSpeed"`
			WindDirection    string `json:"windDirection"`
			ShortForecast    string `json:"shortForecast"`
			DetailedForecast string `json:"detailedForecast"`
		} `json:"periods"`
	} `json:"properties"`
}
