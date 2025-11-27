// Comprehensive test of all functionality
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ngmaloney/mariner-tui/internal/models"
	"github.com/ngmaloney/mariner-tui/internal/noaa"
	"github.com/ngmaloney/mariner-tui/internal/ports"
)

func main() {
	client := ports.NewNOAAStationClient()
	ctx := context.Background()

	// Search for 02633
	fmt.Println("=== Searching for ZIP code 02633 ===")
	results, err := client.SearchByLocation(ctx, "02633")
	if err != nil {
		log.Fatalf("Error searching: %v", err)
	}

	var chathamStation *models.Port
	for _, station := range results {
		if station.State == "MA" {
			chathamStation = &station
			break
		}
	}

	if chathamStation == nil {
		log.Fatal("No Chatham, MA station found")
	}

	fmt.Printf("\nStation: %s\n", chathamStation.Name)
	fmt.Printf("Station ID: %s\n", chathamStation.ID)
	fmt.Printf("Tide Station ID: %s\n", chathamStation.TideStation)
	fmt.Printf("Marine Zone: %s\n", chathamStation.MarineZone)
	fmt.Printf("Coordinates: %.4f, %.4f\n\n", chathamStation.Latitude, chathamStation.Longitude)

	// Test TIDES
	fmt.Println("=== Testing TIDE DATA ===")
	tideClient := noaa.NewTideClient()
	startDate := time.Now()
	endDate := startDate.Add(3 * 24 * time.Hour)

	tides, err := tideClient.GetTidePredictions(ctx, chathamStation.TideStation, startDate, endDate)
	if err != nil {
		fmt.Printf("❌ TIDE ERROR: %v\n\n", err)
	} else {
		fmt.Printf("✓ Found %d tide events\n", len(tides.Events))
		if len(tides.Events) > 0 {
			fmt.Println("Next 5 tides:")
			for i := 0; i < 5 && i < len(tides.Events); i++ {
				event := tides.Events[i]
				fmt.Printf("  %s: %s %.1f ft\n",
					event.Time.Format("Mon Jan 2 3:04 PM"),
					event.Type,
					event.Height)
			}
		}
		fmt.Println()
	}

	// Test MARINE FORECAST
	fmt.Println("=== Testing MARINE FORECAST ===")
	weatherClient := noaa.NewWeatherClient()

	if chathamStation.MarineZone == "" {
		fmt.Println("❌ No marine zone - cannot get marine forecast")
	} else {
		conditions, forecast, err := weatherClient.GetMarineForecastByZone(ctx, chathamStation.MarineZone)
		if err != nil {
			fmt.Printf("❌ FORECAST ERROR: %v\n\n", err)
		} else {
			fmt.Println("✓ Current Conditions:")
			fmt.Printf("  Location: %s\n", conditions.Location)
			fmt.Printf("  Temperature: %.1f°F\n", conditions.Temperature)
			fmt.Printf("  Wind: %s %.0f-%.0f kt", conditions.Wind.Direction, conditions.Wind.SpeedMin, conditions.Wind.SpeedMax)
			if conditions.Wind.HasGust {
				fmt.Printf(", gusts %.0f kt", conditions.Wind.GustSpeed)
			}
			fmt.Println()
			fmt.Printf("  Seas: %.0f-%.0f ft\n", conditions.Seas.HeightMin, conditions.Seas.HeightMax)

			if len(conditions.Seas.Components) > 0 {
				fmt.Println("  Wave Detail:")
				for _, wave := range conditions.Seas.Components {
					fmt.Printf("    %s %.0f ft at %d seconds\n", wave.Direction, wave.Height, wave.Period)
				}
			}

			fmt.Printf("\n✓ Forecast: %d periods\n", len(forecast.Periods))
			if len(forecast.Periods) < 6 {
				fmt.Printf("  ⚠️  WARNING: Only %d periods, expected 6+ for 3-day forecast\n", len(forecast.Periods))
			}

			fmt.Println("\nFirst 3 periods:")
			for i := 0; i < 3 && i < len(forecast.Periods); i++ {
				period := forecast.Periods[i]
				fmt.Printf("\n  %s:\n", period.PeriodName)
				fmt.Printf("    Wind: %s %.0f-%.0f kt\n", period.Wind.Direction, period.Wind.SpeedMin, period.Wind.SpeedMax)
				fmt.Printf("    Seas: %.0f-%.0f ft\n", period.Seas.HeightMin, period.Seas.HeightMax)
			}
		}
	}

	// Test ALERTS
	fmt.Println("\n=== Testing ALERTS ===")
	alertClient := noaa.NewAlertClient()

	if chathamStation.MarineZone != "" {
		alerts, err := alertClient.GetActiveAlertsByZone(ctx, chathamStation.MarineZone)
		if err != nil {
			fmt.Printf("❌ ALERT ERROR: %v\n", err)
		} else {
			fmt.Printf("✓ Found %d active alerts\n", len(alerts.Alerts))
			for _, alert := range alerts.Alerts {
				fmt.Printf("\n  ⚠️  %s (Severity: %v)\n", alert.Event, alert.Severity)
				fmt.Printf("      %s\n", alert.Headline)
			}
		}
	}

	fmt.Println("\n=== SUMMARY ===")
	fmt.Println("Check the output above for ❌ errors")
}
