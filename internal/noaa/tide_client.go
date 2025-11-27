package noaa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ngmaloney/mariner-tui/internal/models"
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
	params.Add("application", "MarinerTUI")

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
