// Test program to verify marine zone forecasts work
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ngmaloney/mariner-tui/internal/models"
	"github.com/ngmaloney/mariner-tui/internal/noaa"
	"github.com/ngmaloney/mariner-tui/internal/ports"
)

func main() {
	client := ports.NewNOAAStationClient()
	ctx := context.Background()

	// Search for 02633
	fmt.Println("Searching for ZIP code 02633...")
	results, err := client.SearchByLocation(ctx, "02633")
	if err != nil {
		log.Fatalf("Error searching: %v", err)
	}

	// Find first Chatham, MA station
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

	fmt.Printf("\nStation: %s (ID: %s)\n", chathamStation.Name, chathamStation.ID)
	fmt.Printf("Marine Zone: %s\n", chathamStation.MarineZone)

	if chathamStation.MarineZone == "" {
		log.Fatal("ERROR: Marine zone not populated!")
	}

	// Test marine forecast
	fmt.Println("\n=== Testing Marine Forecast ===")
	weatherClient := noaa.NewWeatherClient()
	conditions, forecast, err := weatherClient.GetMarineForecastByZone(ctx, chathamStation.MarineZone)
	if err != nil {
		log.Fatalf("Marine forecast error: %v", err)
	}

	fmt.Printf("\nCurrent Conditions:\n")
	fmt.Printf("Wind: %s %.0f-%.0f kt", conditions.Wind.Direction, conditions.Wind.SpeedMin, conditions.Wind.SpeedMax)
	if conditions.Wind.HasGust {
		fmt.Printf(", gusts %.0f kt", conditions.Wind.GustSpeed)
	}
	fmt.Println()
	fmt.Printf("Seas: %.0f-%.0f ft\n", conditions.Seas.HeightMin, conditions.Seas.HeightMax)
	if len(conditions.Seas.Components) > 0 {
		fmt.Println("Wave Detail:")
		for _, wave := range conditions.Seas.Components {
			fmt.Printf("  %s %.0f ft at %d seconds\n", wave.Direction, wave.Height, wave.Period)
		}
	}

	fmt.Printf("\nForecast periods: %d\n", len(forecast.Periods))
	if len(forecast.Periods) > 0 {
		fmt.Println("First period:")
		fmt.Printf("  %s\n", forecast.Periods[0].PeriodName)
		if len(forecast.Periods[0].RawText) > 0 {
			maxLen := len(forecast.Periods[0].RawText)
			if maxLen > 100 {
				maxLen = 100
			}
			fmt.Printf("  %s\n", forecast.Periods[0].RawText[:maxLen])
		}
	}

	// Test alerts
	fmt.Println("\n=== Testing Alerts ===")
	alertClient := noaa.NewAlertClient()
	alerts, err := alertClient.GetActiveAlertsByZone(ctx, chathamStation.MarineZone)
	if err != nil {
		log.Fatalf("Alert error: %v", err)
	}

	fmt.Printf("Found %d active alerts for zone %s\n", len(alerts.Alerts), chathamStation.MarineZone)
	for _, alert := range alerts.Alerts {
		fmt.Printf("\n⚠️  %s\n", alert.Event)
		fmt.Printf("   Headline: %s\n", alert.Headline)
		fmt.Printf("   Severity: %v\n", alert.Severity)
	}

	if len(alerts.Alerts) == 0 {
		fmt.Println("✓ No active alerts (good!)")
	}
}
