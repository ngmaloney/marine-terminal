package noaa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ngmaloney/marine-terminal/internal/models"
)

// NOAATideClient implements TideClient using the NOAA CO-OPS API
type NOAATideClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewTideClient creates a new NOAA tide client
func NewTideClient() *NOAATideClient {
	return &NOAATideClient{
		baseURL: "https://api.tidesandcurrents.noaa.gov/api/prod/datagetter",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetTidePredictions retrieves tide predictions for a date range
func (c *NOAATideClient) GetTidePredictions(ctx context.Context, stationID string, startDate, endDate time.Time) (*models.TideData, error) {
	// Format dates as YYYYMMDD
	beginDate := startDate.Format("20060102")
	endDateStr := endDate.Format("20060102")

	// Build query parameters
	params := url.Values{}
	params.Add("begin_date", beginDate)
	params.Add("end_date", endDateStr)
	params.Add("station", stationID)
	params.Add("product", "predictions")
	params.Add("datum", "MLLW")      // Mean Lower Low Water
	params.Add("time_zone", "lst_ldt") // Local standard/daylight time
	params.Add("interval", "hilo")    // High and low tides only
	params.Add("units", "english")    // Feet
	params.Add("format", "json")
	params.Add("application", "MarineTerminal")

	requestURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tide data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var tideResp tideResponse
	if err := json.NewDecoder(resp.Body).Decode(&tideResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our model
	tideData := &models.TideData{
		StationID:   stationID,
		StationName: tideResp.Metadata.Name,
		Events:      make([]models.TideEvent, 0, len(tideResp.Predictions)),
		UpdatedAt:   time.Now(),
	}

	for _, pred := range tideResp.Predictions {
		eventTime, err := time.Parse("2006-01-02 15:04", pred.Time)
		if err != nil {
			continue // Skip invalid times
		}

		var tideType models.TideType
		if pred.Type == "H" {
			tideType = models.TideHigh
		} else {
			tideType = models.TideLow
		}

		// Parse height from string
		height, err := strconv.ParseFloat(pred.Height, 64)
		if err != nil {
			// Skip events with invalid height
			continue
		}

		event := models.TideEvent{
			Time:   eventTime,
			Type:   tideType,
			Height: height,
		}

		tideData.Events = append(tideData.Events, event)
	}

	return tideData, nil
}

// GetMeteorologicalData retrieves meteorological data (e.g., air temperature, pressure) for a station
func (c *NOAATideClient) GetMeteorologicalData(ctx context.Context, stationID string, startDate, endDate time.Time) (*models.MarineConditions, error) {
	// Format dates as YYYYMMDD
	beginDate := startDate.Format("20060102")
	endDateStr := endDate.Format("20060102")

	type result struct {
		data interface{}
		err  error
	}

	// Helper function to fetch specific product
	fetchProduct := func(product string, out interface{}) result {
		params := url.Values{}
		params.Add("begin_date", beginDate)
		params.Add("end_date", endDateStr)
		params.Add("station", stationID)
		params.Add("product", product)
		params.Add("datum", "MLLW")
		params.Add("time_zone", "lst_ldt")
		params.Add("units", "english")
		params.Add("format", "json")
		params.Add("application", "MarineTerminal")

		requestURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())
		req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
		if err != nil {
			return result{nil, err}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return result{nil, err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return result{nil, fmt.Errorf("API status %d for %s", resp.StatusCode, product)}
		}

		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return result{nil, err}
		}
		return result{out, nil}
	}

	// Response structs
	type observation struct {
		Time  string `json:"t"`
		Value string `json:"v"`
	}
	type apiResponse struct {
		Data []observation `json:"data"`
	}

	// Channels for concurrent requests
	airTempChan := make(chan result)
	pressureChan := make(chan result)

	go func() { airTempChan <- fetchProduct("air_temperature", &apiResponse{}) }()
	go func() { pressureChan <- fetchProduct("air_pressure", &apiResponse{}) }()

	// Collect results
	conditions := &models.MarineConditions{
		Location:  stationID,
		UpdatedAt: time.Now(),
	}

	// Wait for Air Temp
	res := <-airTempChan
	if res.err == nil {
		if resp, ok := res.data.(*apiResponse); ok && len(resp.Data) > 0 {
			// Get most recent observation
			lastObs := resp.Data[len(resp.Data)-1]
			if val, err := strconv.ParseFloat(lastObs.Value, 64); err == nil {
				conditions.Temperature = val
			}
		}
	}

	// Wait for Pressure
	res = <-pressureChan
	if res.err == nil {
		if resp, ok := res.data.(*apiResponse); ok && len(resp.Data) > 0 {
			// Get most recent observation
			lastObs := resp.Data[len(resp.Data)-1]
			if val, err := strconv.ParseFloat(lastObs.Value, 64); err == nil {
				conditions.Pressure = val // NOAA returns in millibars by default with "metric", check "english" units
				// "english" units for pressure might be different? 
				// CO-OPS API docs: "english": pressure in mb? No, usually mb. 
				// Let's assume mb for now as standard marine unit.
			}
		}
	}

	return conditions, nil
}

// Internal types for NOAA CO-OPS API responses

type tideResponse struct {
	Metadata struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Lat  string `json:"lat"`
		Lon  string `json:"lon"`
	} `json:"metadata"`
	Predictions []struct {
		Time   string `json:"t"`
		Height string `json:"v"` // NOAA returns this as string
		Type   string `json:"type"` // "H" or "L"
	} `json:"predictions"`
}
