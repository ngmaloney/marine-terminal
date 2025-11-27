package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ngmaloney/mariner-tui/internal/models"
)

// renderWeatherPane renders the weather information pane
func (m Model) renderWeatherPane(width int) string {
	// Calculate content width (total width - border - padding)
	// Border: 2 chars, Padding: 4 chars = 6 total overhead
	contentWidth := width - 6
	if contentWidth < 20 {
		contentWidth = 20
	}

	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render("Weather"))
	content.WriteString("\n\n")

	// Check if we have weather data
	if m.weather == nil && m.forecast == nil {
		content.WriteString(mutedStyle.Render("No weather data available"))
		return paneStyle.Width(width).Render(content.String())
	}

	// Current conditions (from first forecast period)
	if m.weather != nil {
		// Get the period name from the first forecast period if available
		periodName := "Current Conditions"
		if m.forecast != nil && len(m.forecast.Periods) > 0 {
			periodName = m.forecast.Periods[0].PeriodName
		}

		// Wrap content to fit within pane
		wrappedStyle := lipgloss.NewStyle().Width(contentWidth)

		content.WriteString(labelStyle.Render(periodName))
		content.WriteString("\n")
		content.WriteString(wrappedStyle.Render(m.weather.Conditions))
		content.WriteString("\n\n")

		// Only show temperature if it's available (> 0)
		if m.weather.Temperature > 0 {
			content.WriteString(labelStyle.Render("Temperature: "))
			content.WriteString(valueStyle.Render(fmt.Sprintf("%.0f°F", m.weather.Temperature)))
			content.WriteString("\n\n")
		}

		// Wind
		if m.weather.Wind.Direction != "" {
			content.WriteString(labelStyle.Render("Wind: "))
			windStr := formatWind(m.weather.Wind)
			content.WriteString(valueStyle.Render(windStr))
			content.WriteString("\n\n")
		}

		// Seas
		if m.weather.Seas.HeightMin > 0 || m.weather.Seas.HeightMax > 0 {
			content.WriteString(labelStyle.Render("Seas: "))
			seasStr := formatSeas(m.weather.Seas)
			content.WriteString(valueStyle.Render(seasStr))
			content.WriteString("\n")

			// Wave detail
			if len(m.weather.Seas.Components) > 0 {
				content.WriteString(labelStyle.Render("Wave Detail:\n"))
				for _, wave := range m.weather.Seas.Components {
					waveStr := fmt.Sprintf("  %s %.0f ft at %d sec", wave.Direction, wave.Height, wave.Period)
					content.WriteString(valueStyle.Render(waveStr))
					content.WriteString("\n")
				}
			}
		}
	}

	// 3-Day Forecast (skip first period since it's shown above as current)
	if m.forecast != nil && len(m.forecast.Periods) > 1 {
		content.WriteString("\n")
		content.WriteString(labelStyle.Render("3-Day Forecast"))
		content.WriteString("\n")

		// Wrap content to fit within pane
		wrappedStyle := lipgloss.NewStyle().Width(contentWidth)

		// Show periods 1-6 for proper 3-day coverage (skip period 0 shown as current)
		startIdx := 1
		maxPeriods := 6
		if len(m.forecast.Periods)-startIdx < maxPeriods {
			maxPeriods = len(m.forecast.Periods) - startIdx
		}

		for i := 0; i < maxPeriods; i++ {
			period := m.forecast.Periods[startIdx+i]
			content.WriteString(fmt.Sprintf("\n%s\n", valueStyle.Bold(true).Render(period.PeriodName)))

			// Show wind and seas for each period - wrapped to content width
			if period.Wind.Direction != "" {
				windText := fmt.Sprintf("Wind: %s", formatWind(period.Wind))
				content.WriteString(wrappedStyle.Render(windText))
				content.WriteString("\n")
			}
			if period.Seas.HeightMin > 0 || period.Seas.HeightMax > 0 {
				seasText := fmt.Sprintf("Seas: %s", formatSeas(period.Seas))
				content.WriteString(wrappedStyle.Render(seasText))
				content.WriteString("\n")
			}
		}
	}

	return paneStyle.Width(width).Render(content.String())
}

// formatWind formats wind data for display
func formatWind(wind models.WindData) string {
	if wind.SpeedMin == wind.SpeedMax {
		if wind.HasGust {
			return fmt.Sprintf("%s %0.f kt, gusts %0.f kt",
				wind.Direction, wind.SpeedMin, wind.GustSpeed)
		}
		return fmt.Sprintf("%s %.0f kt", wind.Direction, wind.SpeedMin)
	}

	if wind.HasGust {
		return fmt.Sprintf("%s %.0f-%.0f kt, gusts %.0f kt",
			wind.Direction, wind.SpeedMin, wind.SpeedMax, wind.GustSpeed)
	}
	return fmt.Sprintf("%s %.0f-%.0f kt", wind.Direction, wind.SpeedMin, wind.SpeedMax)
}

// formatSeas formats sea state for display
func formatSeas(seas models.SeaState) string {
	if seas.HeightMin == seas.HeightMax {
		return fmt.Sprintf("%.0f ft", seas.HeightMin)
	}
	return fmt.Sprintf("%.0f-%.0f ft", seas.HeightMin, seas.HeightMax)
}
