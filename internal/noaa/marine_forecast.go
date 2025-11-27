package noaa

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ngmaloney/mariner-tui/internal/models"
)

// GetMarineForecastByZone retrieves marine forecast for a given zone
func (c *NOAAWeatherClient) GetMarineForecastByZone(ctx context.Context, marineZone string) (*models.MarineConditions, *models.ThreeDayForecast, error) {
	if marineZone == "" {
		return nil, nil, fmt.Errorf("marine zone is required")
	}

	// NOAA's JSON API doesn't support marine forecasts, use text products instead
	// Format: https://tgftp.nws.noaa.gov/data/forecasts/marine/coastal/an/anz254.txt
	zoneType := determineZoneType(marineZone)
	url := fmt.Sprintf("https://tgftp.nws.noaa.gov/data/forecasts/marine/%s/%s/%s.txt",
		zoneType, getZonePrefix(marineZone), strings.ToLower(marineZone))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching forecast: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("marine text product returned status %d for zone %s", resp.StatusCode, marineZone)
	}

	// Read the full text response
	textBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}

	// Parse the marine text forecast
	return parseMarineTextProduct(string(textBytes), marineZone)
}

// determineZoneType returns the forecast type based on zone prefix
func determineZoneType(zone string) string {
	zone = strings.ToUpper(zone)
	if strings.HasPrefix(zone, "AN") || strings.HasPrefix(zone, "GM") {
		return "coastal"
	}
	return "offshore"
}

// getZonePrefix returns the two-letter prefix for the zone directory
func getZonePrefix(zone string) string {
	if len(zone) < 2 {
		return "an"
	}
	return strings.ToLower(zone[:2])
}

// parseMarineTextProduct parses NOAA's marine text product format
func parseMarineTextProduct(text, zone string) (*models.MarineConditions, *models.ThreeDayForecast, error) {
	// Split by period markers
	lines := strings.Split(text, "\n.")
	var periods []struct {
		name string
		text string
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Format is "PERIOD NAME...forecast text"
		parts := strings.SplitN(line, "...", 2)
		if len(parts) == 2 {
			periodName := strings.TrimSpace(parts[0])
			periodText := strings.TrimSpace(parts[1])

			// Skip alert headers (they start with dots and are all caps)
			if strings.HasPrefix(periodName, ".") {
				continue
			}

			// Valid forecast periods: THIS AFTERNOON, TONIGHT, FRI, SAT, SUN, etc.
			// Skip if it looks like an alert (has words like ADVISORY, WARNING)
			if strings.Contains(strings.ToUpper(periodName), "ADVISORY") ||
				strings.Contains(strings.ToUpper(periodName), "WARNING") ||
				strings.Contains(strings.ToUpper(periodName), "WATCH") {
				continue
			}

			periods = append(periods, struct {
				name string
				text string
			}{
				name: periodName,
				text: periodText,
			})
		}
	}

	if len(periods) == 0 {
		return nil, nil, fmt.Errorf("no forecast periods found in text product")
	}

	// Parse first period for current conditions
	conditions := parseMarineForecast(periods[0].text, zone)

	// Parse all periods for forecast
	forecast := &models.ThreeDayForecast{
		Periods:   make([]models.MarineForecast, 0, len(periods)),
		UpdatedAt: time.Now(),
	}

	for _, period := range periods {
		marineCond := parseMarineForecast(period.text, zone)
		forecast.Periods = append(forecast.Periods, models.MarineForecast{
			PeriodName:  period.name,
			Conditions:  marineCond.Conditions,
			Wind:        marineCond.Wind,
			Seas:        marineCond.Seas,
			Temperature: marineCond.Temperature,
			RawText:     period.text,
		})
	}

	return conditions, forecast, nil
}

// parseMarineForecast parses a NOAA marine forecast text into structured data
func parseMarineForecast(forecastText, zone string) *models.MarineConditions {
	conditions := &models.MarineConditions{
		Location:  zone,
		UpdatedAt: time.Now(),
	}

	// Parse wind (e.g., "W winds 15 to 20 kt with gusts up to 30 kt")
	windRegex := regexp.MustCompile(`(?i)([NESW]+)\s+(?:winds?\s+)?(\d+)(?:\s+to\s+(\d+))?\s*kt`)
	if match := windRegex.FindStringSubmatch(forecastText); len(match) > 0 {
		direction := match[1]
		speedMin, _ := strconv.ParseFloat(match[2], 64)
		speedMax := speedMin
		if len(match) > 3 && match[3] != "" {
			speedMax, _ = strconv.ParseFloat(match[3], 64)
		}

		conditions.Wind = models.WindData{
			Direction: direction,
			SpeedMin:  speedMin,
			SpeedMax:  speedMax,
			RawText:   match[0],
		}

		// Check for gusts
		gustRegex := regexp.MustCompile(`(?i)gusts?\s+(?:up\s+to\s+)?(\d+)\s*kt`)
		if gustMatch := gustRegex.FindStringSubmatch(forecastText); len(gustMatch) > 0 {
			gust, _ := strconv.ParseFloat(gustMatch[1], 64)
			conditions.Wind.GustSpeed = gust
			conditions.Wind.HasGust = true
		}
	}

	// Parse seas (e.g., "Seas 5 to 7 ft" or "Waves 3 to 5 ft")
	seasRegex := regexp.MustCompile(`(?i)(?:seas|waves)\s+(\d+)(?:\s+to\s+(\d+))?\s*ft`)
	if match := seasRegex.FindStringSubmatch(forecastText); len(match) > 0 {
		heightMin, _ := strconv.ParseFloat(match[1], 64)
		heightMax := heightMin
		if len(match) > 2 && match[2] != "" {
			heightMax, _ = strconv.ParseFloat(match[2], 64)
		}

		conditions.Seas = models.SeaState{
			HeightMin:  heightMin,
			HeightMax:  heightMax,
			Components: []models.WaveComponent{},
			RawText:    match[0],
		}
	}

	// Parse wave components (e.g., "S 5 ft at 8 seconds")
	waveRegex := regexp.MustCompile(`(?i)([NESW]+)\s+(\d+)\s*ft\s+at\s+(\d+)\s+seconds?`)
	for _, match := range waveRegex.FindAllStringSubmatch(forecastText, -1) {
		if len(match) > 0 {
			height, _ := strconv.ParseFloat(match[2], 64)
			period, _ := strconv.Atoi(match[3])

			conditions.Seas.Components = append(conditions.Seas.Components, models.WaveComponent{
				Direction: match[1],
				Height:    height,
				Period:    period,
			})
		}
	}

	// Store the full forecast text
	conditions.Conditions = forecastText

	return conditions
}
