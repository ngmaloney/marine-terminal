package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ngmaloney/marine-terminal/internal/ui"
)

func main() {
	stationCode := flag.String("station", "", "Specify a marine station code to load directly (requires --location) (e.g., ANZ251)")
	location := flag.String("location", "", "Specify location for station lookup (zipcode or city, state)")
	portName := flag.String("port", "", "Name of a saved port to load directly")
	flag.Parse()

	// Validation logic: if station is provided, location must be too
	if *stationCode != "" && *location == "" {
		fmt.Println("Error: --station requires --location to determine the nearest tide station.")
		os.Exit(1)
	}

	p := tea.NewProgram(ui.NewModel(*stationCode, *location, *portName), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running application: %v\n", err)
		os.Exit(1)
	}
}