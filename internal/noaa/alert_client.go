package noaa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ngmaloney/marine-terminal/internal/models"
)

const cacheDuration = 15 * time.Minute

type cacheEntry struct {
	data      *models.AlertData
	fetchedAt time.Time
}

// NOAAAlertClient implements AlertClient using the NOAA Weather API
type NOAAAlertClient struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
	cache      map[string]cacheEntry
	mu         sync.RWMutex
}

// NewAlertClient creates a new NOAA alert client
func NewAlertClient() *NOAAAlertClient {
	return &NOAAAlertClient{
		baseURL: "https://api.weather.gov",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "MarineTerminal/1.0 (github.com/ngmaloney/marine-terminal)",
		cache:      make(map[string]cacheEntry),
	}
}

// GetActiveAlerts retrieves active marine alerts for a location
func (c *NOAAAlertClient) GetActiveAlerts(ctx context.Context, lat, lon float64) (*models.AlertData, error) {
	// Query alerts by point
	url := fmt.Sprintf("%s/alerts/active?point=%.4f,%.4f", c.baseURL, lat, lon)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch alerts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var alertResp alertResponse
	if err := json.NewDecoder(resp.Body).Decode(&alertResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our model
	alertData := &models.AlertData{
		Alerts:    make([]models.Alert, 0),
		UpdatedAt: time.Now(),
	}

	for _, feature := range alertResp.Features {
		props := feature.Properties

		// Parse times
		onset, _ := time.Parse(time.RFC3339, props.Onset)
		expires, _ := time.Parse(time.RFC3339, props.Expires)

		// Map severity
		severity := mapSeverity(props.Severity)

		// Extract affected areas
		areas := make([]string, 0)
		if props.AreaDesc != "" {
			areas = append(areas, props.AreaDesc)
		}

		alert := models.Alert{
			ID:          props.ID,
			Event:       props.Event,
			Headline:    props.Headline,
			Description: props.Description,
			Severity:    severity,
			Urgency:     props.Urgency,
			Certainty:   props.Certainty,
			Onset:       onset,
			Expires:     expires,
			Areas:       areas,
			Instruction: props.Instruction,
		}

		// Only include marine alerts or all if no filter
		if alert.IsMarine() {
			alertData.Alerts = append(alertData.Alerts, alert)
		}
	}

	return alertData, nil
}

// GetActiveAlertsByZone retrieves active alerts for a specific marine zone
func (c *NOAAAlertClient) GetActiveAlertsByZone(ctx context.Context, marineZone string) (*models.AlertData, error) {
	// Check cache first
	c.mu.RLock()
	entry, ok := c.cache[marineZone]
	c.mu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < cacheDuration {
		return entry.data, nil
	}

	// Query alerts by zone
	url := fmt.Sprintf("%s/alerts/active?zone=%s", c.baseURL, marineZone)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch alerts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var alertResp alertResponse
	if err := json.NewDecoder(resp.Body).Decode(&alertResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our model
	alertData := &models.AlertData{
		Alerts:    make([]models.Alert, 0),
		UpdatedAt: time.Now(),
	}

	for _, feature := range alertResp.Features {
		props := feature.Properties

		// Parse times
		onset, _ := time.Parse(time.RFC3339, props.Onset)
		expires, _ := time.Parse(time.RFC3339, props.Expires)

		// Map severity
		severity := mapSeverity(props.Severity)

		// Extract affected areas
		areas := make([]string, 0)
		if props.AreaDesc != "" {
			areas = append(areas, props.AreaDesc)
		}

		alert := models.Alert{
			ID:          props.ID,
			Event:       props.Event,
			Headline:    props.Headline,
			Description: props.Description,
			Severity:    severity,
			Urgency:     props.Urgency,
			Certainty:   props.Certainty,
			Onset:       onset,
			Expires:     expires,
			Areas:       areas,
			Instruction: props.Instruction,
		}

		// Include all alerts for this zone (they should all be marine)
		alertData.Alerts = append(alertData.Alerts, alert)
	}

	// Store in cache
	c.mu.Lock()
	c.cache[marineZone] = cacheEntry{data: alertData, fetchedAt: time.Now()}
	c.mu.Unlock()

	return alertData, nil
}

func mapSeverity(s string) models.AlertSeverity {
	switch s {
	case "Extreme":
		return models.SeverityExtreme
	case "Severe":
		return models.SeveritySevere
	case "Moderate":
		return models.SeverityModerate
	case "Minor":
		return models.SeverityMinor
	default:
		return models.SeverityUnknown
	}
}

// Internal types for NOAA Alert API responses

type alertResponse struct {
	Features []struct {
		ID         string `json:"id"`
		Properties struct {
			ID          string `json:"id"`
			Event       string `json:"event"`
			Headline    string `json:"headline"`
			Description string `json:"description"`
			Severity    string `json:"severity"`
			Urgency     string `json:"urgency"`
			Certainty   string `json:"certainty"`
			Onset       string `json:"onset"`
			Expires     string `json:"expires"`
			AreaDesc    string `json:"areaDesc"`
			Instruction string `json:"instruction"`
		} `json:"properties"`
	} `json:"features"`
}
