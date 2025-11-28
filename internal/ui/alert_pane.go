package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/ngmaloney/mariner-tui/internal/models"
)

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
