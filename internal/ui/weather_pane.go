package ui

import (
	"fmt"

	"github.com/ngmaloney/mariner-tui/internal/models"
)

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
