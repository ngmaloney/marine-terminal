package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ngmaloney/mariner-tui/internal/models"
)

// renderWeatherSimple renders weather without borders or width constraints
func (m Model) renderWeatherSimple() string {
	if m.weather == nil && m.forecast == nil {
		return mutedStyle.Render("No weather data available")
	}

	var lines []string

	// Current conditions
	if m.weather != nil && m.forecast != nil && len(m.forecast.Periods) > 0 {
		lines = append(lines, valueStyle.Bold(true).Render(m.forecast.Periods[0].PeriodName))

		if m.weather.Wind.Direction != "" {
			lines = append(lines, fmt.Sprintf("Wind: %s", formatWind(m.weather.Wind)))
		}

		if m.weather.Seas.HeightMin > 0 || m.weather.Seas.HeightMax > 0 {
			lines = append(lines, fmt.Sprintf("Seas: %s", formatSeas(m.weather.Seas)))
		}

		if len(m.weather.Seas.Components) > 0 {
			for _, wave := range m.weather.Seas.Components {
				lines = append(lines, fmt.Sprintf("  %s %.0f ft at %d sec", wave.Direction, wave.Height, wave.Period))
			}
		}
	}

	// Forecast
	if m.forecast != nil && len(m.forecast.Periods) > 1 {
		lines = append(lines, "", labelStyle.Render("Forecast:"))

		maxPeriods := 6
		if len(m.forecast.Periods)-1 < maxPeriods {
			maxPeriods = len(m.forecast.Periods) - 1
		}

		for i := 1; i <= maxPeriods; i++ {
			period := m.forecast.Periods[i]
			lines = append(lines, fmt.Sprintf("  %s: %s, Seas %s",
				period.PeriodName,
				formatWind(period.Wind),
				formatSeas(period.Seas)))
		}
	}

	return strings.Join(lines, "\n")
}

// renderTideSimple renders tides without borders or width constraints
func (m Model) renderTideSimple() string {
	if m.tides == nil || len(m.tides.Events) == 0 {
		return mutedStyle.Render("No tide data available")
	}

	var lines []string
	today := time.Now()

	for day := 0; day < 3; day++ {
		date := today.AddDate(0, 0, day)
		events := m.tides.GetEventsForDay(date)

		if len(events) == 0 {
			continue
		}

		var dayLabel string
		if day == 0 {
			dayLabel = "Today"
		} else if day == 1 {
			dayLabel = "Tomorrow"
		} else {
			dayLabel = date.Format("Monday")
		}

		lines = append(lines, labelStyle.Render(fmt.Sprintf("%s (%s):", dayLabel, date.Format("Jan 2"))))

		for _, event := range events {
			typeStr := "Low"
			if event.Type == models.TideHigh {
				typeStr = "High"
			}
			lines = append(lines, fmt.Sprintf("  %s - %s %.1f ft",
				event.Time.Format("3:04 PM"),
				typeStr,
				event.Height))
		}
	}

	return strings.Join(lines, "\n")
}

// renderAlertSimple renders alerts without borders or width constraints
func (m Model) renderAlertSimple() string {
	if m.alerts == nil {
		return mutedStyle.Render("No alert data available")
	}

	activeAlerts := make([]models.Alert, 0)
	for _, alert := range m.alerts.Alerts {
		if alert.IsActive() && alert.IsMarine() {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	if len(activeAlerts) == 0 {
		return successStyle.Render("✓ No active marine alerts")
	}

	var lines []string
	for _, alert := range activeAlerts {
		alertStyle := getAlertStyle(alert.Severity)
		lines = append(lines,
			alertStyle.Render(fmt.Sprintf("⚠  %s", alert.Event)),
			fmt.Sprintf("   %s", alert.Headline),
			fmt.Sprintf("   Expires: %s", alert.Expires.Format("Jan 2, 3:04 PM")),
			"",
		)
	}

	return strings.Join(lines, "\n")
}
