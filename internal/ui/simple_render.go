package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ngmaloney/mariner-tui/internal/models"
)

// renderWeatherSimple renders weather with nice styling
func (m Model) renderWeatherSimple() string {
	if m.weather == nil && m.forecast == nil {
		return mutedStyle.Render("No weather data available")
	}

	var lines []string

	// Current conditions
	if m.weather != nil && m.forecast != nil && len(m.forecast.Periods) > 0 {
		periodStyle := lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)
		lines = append(lines, periodStyle.Render(m.forecast.Periods[0].PeriodName))

		if m.weather.Wind.Direction != "" {
			windLabel := labelStyle.Render("Wind: ")
			windValue := valueStyle.Render(formatWind(m.weather.Wind))
			lines = append(lines, windLabel+windValue)
		}

		if m.weather.Seas.HeightMin > 0 || m.weather.Seas.HeightMax > 0 {
			seasLabel := labelStyle.Render("Seas: ")
			seasValue := valueStyle.Render(formatSeas(m.weather.Seas))
			lines = append(lines, seasLabel+seasValue)
		}

		if len(m.weather.Seas.Components) > 0 {
			for _, wave := range m.weather.Seas.Components {
				waveText := fmt.Sprintf("  %s %.0f ft at %d sec", wave.Direction, wave.Height, wave.Period)
				lines = append(lines, mutedStyle.Render(waveText))
			}
		}
	}

	// Forecast
	if m.forecast != nil && len(m.forecast.Periods) > 1 {
		lines = append(lines, "", labelStyle.Render("📅 3-Day Forecast:"))

		maxPeriods := 6
		if len(m.forecast.Periods)-1 < maxPeriods {
			maxPeriods = len(m.forecast.Periods) - 1
		}

		for i := 1; i <= maxPeriods; i++ {
			period := m.forecast.Periods[i]
			periodName := valueStyle.Render(period.PeriodName + ":")
			windSeas := mutedStyle.Render(fmt.Sprintf("%s, Seas %s",
				formatWind(period.Wind),
				formatSeas(period.Seas)))
			lines = append(lines, fmt.Sprintf("  %s %s", periodName, windSeas))
		}
	}

	return strings.Join(lines, "\n")
}


// renderAlertSimple renders alerts with nice styling
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
		return successStyle.Bold(true).Render("✓ No active marine alerts")
	}

	var lines []string
	for i, alert := range activeAlerts {
		if i > 0 {
			lines = append(lines, "")
		}

		alertStyle := getAlertStyle(alert.Severity)
		lines = append(lines, alertStyle.Render(fmt.Sprintf("️%s", alert.Event)))
		lines = append(lines, valueStyle.Render(alert.Headline))

		expiresLabel := labelStyle.Render("Expires: ")
		expiresValue := mutedStyle.Render(alert.Expires.Format("Jan 2, 3:04 PM"))
		lines = append(lines, expiresLabel+expiresValue)
	}

	return strings.Join(lines, "\n")
}
