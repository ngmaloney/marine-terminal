package noaa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// GetMarineZone finds the marine forecast zone for a given location
func GetMarineZone(ctx context.Context, lat, lon float64) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// Query NOAA for zones at this location
	url := fmt.Sprintf("https://api.weather.gov/zones?type=forecast&point=%.4f,%.4f", lat, lon)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "MarinerTUI/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching zones: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var zonesResp struct {
		Features []struct {
			Properties struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"properties"`
		} `json:"features"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&zonesResp); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	// Look for marine zones (typically start with A, P, or G for Atlantic, Pacific, Gulf)
	for _, feature := range zonesResp.Features {
		zoneID := feature.Properties.ID
		zoneName := strings.ToLower(feature.Properties.Name)

		// Extract the zone code from the ID (e.g., "https://api.weather.gov/zones/forecast/ANZ254" -> "ANZ254")
		parts := strings.Split(zoneID, "/")
		if len(parts) > 0 {
			zoneCode := parts[len(parts)-1]

			// Check if this is a marine zone
			if strings.Contains(zoneName, "waters") ||
				strings.Contains(zoneName, "marine") ||
				strings.Contains(zoneName, "offshore") ||
				strings.Contains(zoneName, "coastal") {
				return zoneCode, nil
			}
		}
	}

	return "", fmt.Errorf("no marine zone found for location %.4f, %.4f", lat, lon)
}
