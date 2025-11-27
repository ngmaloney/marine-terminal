package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ngmaloney/mariner-tui/internal/models"
)

// renderTidePane renders the tide information pane
func (m Model) renderTidePane(width int) string {
	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render("Tides"))
	content.WriteString("\n\n")

	// Check if we have tide data
	if m.tides == nil || len(m.tides.Events) == 0 {
		content.WriteString(mutedStyle.Render("No tide data available"))
		return paneStyle.Width(width).Render(content.String())
	}

	// Group tides by day
	today := time.Now()
	for day := 0; day < 3; day++ {
		date := today.AddDate(0, 0, day)
		events := m.tides.GetEventsForDay(date)

		if len(events) == 0 {
			continue
		}

		// Day header
		var dayLabel string
		if day == 0 {
			dayLabel = "Today"
		} else if day == 1 {
			dayLabel = "Tomorrow"
		} else {
			dayLabel = date.Format("Monday")
		}

		content.WriteString(labelStyle.Render(dayLabel))
		content.WriteString(fmt.Sprintf(" %s\n", mutedStyle.Render(date.Format("Jan 2"))))

		// Tide events for this day
		for _, event := range events {
			timeStr := event.Time.Format("3:04 PM")
			typeStr := "Low"
			if event.Type == models.TideHigh {
				typeStr = "High"
			}

			line := fmt.Sprintf("  %s  %s  %.1f ft\n",
				valueStyle.Render(timeStr),
				labelStyle.Width(4).Render(typeStr),
				event.Height)
			content.WriteString(line)
		}
		content.WriteString("\n")
	}

	return paneStyle.Width(width).Render(content.String())
}
