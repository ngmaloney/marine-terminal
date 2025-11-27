package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ngmaloney/mariner-tui/internal/models"
)

// renderAlertPane renders the alerts pane
func (m Model) renderAlertPane(width int) string {
	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render("Alerts"))
	content.WriteString("\n\n")

	// Check if we have alert data
	if m.alerts == nil {
		content.WriteString(mutedStyle.Render("No alert data available"))
		return paneStyle.Width(width).Render(content.String())
	}

	// Filter for active marine alerts only
	activeAlerts := make([]models.Alert, 0)
	for _, alert := range m.alerts.Alerts {
		if alert.IsActive() && alert.IsMarine() {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	if len(activeAlerts) == 0 {
		content.WriteString(successStyle.Render("✓ No active marine alerts"))
		return paneStyle.Width(width).Render(content.String())
	}

	// Calculate content width (accounting for padding)
	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Display alerts
	for i, alert := range activeAlerts {
		if i > 0 {
			content.WriteString("\n")
			content.WriteString(strings.Repeat("─", contentWidth))
			content.WriteString("\n\n")
		}

		// Alert event with severity styling
		eventStyle := getAlertStyle(alert.Severity)
		content.WriteString(eventStyle.Render(alert.Event))
		content.WriteString("\n\n")

		// Headline
		if alert.Headline != "" {
			wrapped := lipgloss.NewStyle().Width(contentWidth).Render(alert.Headline)
			content.WriteString(valueStyle.Render(wrapped))
			content.WriteString("\n\n")
		}

		// Expiration
		expiresStr := alert.Expires.Format("Jan 2, 3:04 PM")
		content.WriteString(labelStyle.Render("Expires: "))
		content.WriteString(mutedStyle.Render(expiresStr))
		content.WriteString("\n")

		// Affected areas
		if len(alert.Areas) > 0 {
			content.WriteString(labelStyle.Render("Areas: "))
			content.WriteString(valueStyle.Render(strings.Join(alert.Areas, ", ")))
			content.WriteString("\n")
		}
	}

	return paneStyle.Width(width).Render(content.String())
}

// getAlertStyle returns the appropriate style for an alert severity
func getAlertStyle(severity models.AlertSeverity) lipgloss.Style {
	switch severity {
	case models.SeverityExtreme:
		return alertExtremeStyle
	case models.SeveritySevere:
		return alertSevereStyle
	case models.SeverityModerate:
		return alertModerateStyle
	case models.SeverityMinor:
		return alertMinorStyle
	default:
		return valueStyle
	}
}
