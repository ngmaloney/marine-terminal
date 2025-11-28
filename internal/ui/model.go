package ui

import (
	"fmt"
	"strings" // Added import for strings

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/ngmaloney/marine-terminal/internal/database"
	"github.com/ngmaloney/marine-terminal/internal/geocoding"
	"github.com/ngmaloney/marine-terminal/internal/models"
	"github.com/ngmaloney/marine-terminal/internal/noaa"
	"github.com/ngmaloney/marine-terminal/internal/stations"
	"github.com/ngmaloney/marine-terminal/internal/zonelookup"
)

// AppState represents the current state of the application
type AppState int

const (
	StateSearch       AppState = iota // Search for location (zipcode/city/state)
	StateZoneList                     // Show list of nearby marine zones
	StateLoading                      // Loading weather/alert data
	StateDisplay                      // Display weather/alerts for selected zone
	StateProvisioning                 // Initial data provisioning (downloading/building DB)
	StateError                        // Error state
)

// ActivePane represents which pane is currently focused
type ActivePane int

const (
	PaneWeather ActivePane = iota
	PaneTides
)

// Model represents the application's state
type Model struct {
	state       AppState
	activePane  ActivePane
	width       int
	height      int
	err         error

	// Search
	searchInput textinput.Model
	geocoder    *geocoding.Geocoder
	searchQuery string // Last search query

	// Location and zones
	location      *geocoding.Location
	zones         []zonelookup.ZoneInfo
	zoneList      list.Model
	selectedZone  *zonelookup.ZoneInfo
	tideStations  []stations.TideStationInfo
	tideStation   *stations.TideStationInfo

	// Charts
	tideChart timeserieslinechart.Model

	// API clients
	weatherClient noaa.WeatherClient
	alertClient   noaa.AlertClient
	tideClient    noaa.TideClient

	// Data
	weather  *models.MarineConditions
	forecast *models.ThreeDayForecast
	alerts   *models.AlertData
	tides    *models.TideData
	tideConditions *models.MarineConditions

	// Loading states
	loadingWeather bool
	loadingAlerts  bool
	loadingTides   bool

	// Provisioning
	spinner           spinner.Model
	provisionStatus   string
	provisionChannels *provisioningStartedMsg

	initialStationCode string // New: for direct loading via CLI arg
}

// NewModel creates a new application model
func NewModel(initialStationCode string) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter zipcode or city, state (e.g. 02633 or Chatham, MA)..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 60

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	tc := timeserieslinechart.New(50, 15)
	
	return Model{
		state:         StateSearch, // Will be checked in Init, potentially overridden
		activePane:    PaneWeather,
		searchInput:   ti,
		geocoder:      geocoding.NewGeocoder(),
		weatherClient: noaa.NewWeatherClient(),
		alertClient:   noaa.NewAlertClient(),
		tideClient:    noaa.NewTideClient(),
		spinner:       s,
		tideChart:     tc,
		initialStationCode: initialStationCode,
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	dbPath := database.DBPath()
	
	// Check if we need to provision the database (marine zones or zipcodes)
	zonesNeeded, err := zonelookup.NeedsProvisioning(dbPath)
	if err != nil {
		return textinput.Blink
	}
	
	zipNeeded, err := geocoding.NeedsProvisioning(dbPath)
	if err != nil {
		return textinput.Blink
	}

	if zonesNeeded || zipNeeded {
		return tea.Batch(m.spinner.Tick, initiateProvisioning())
	}

	// If no provisioning needed, check for direct station load
	if m.initialStationCode != "" {
		m.state = StateLoading
		return tea.Batch(m.spinner.Tick, directLoadStation(m.initialStationCode))
	}

	return textinput.Blink
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle window size
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		// Update zone list size if it exists
		if m.state == StateZoneList {
			m.zoneList.SetSize(msg.Width-4, msg.Height-10)
		}
		return m, nil
	}

	// Handle custom messages
	switch msg := msg.(type) {
	case errMsg:
		m.err = msg.err
		m.state = StateError
		return m, nil

	// Provisioning messages
	case provisioningStartedMsg:
		m.state = StateProvisioning
		m.provisionStatus = "Starting data provisioning..."
		m.provisionChannels = &msg
		return m, tea.Batch(
			waitForProvisionStatus(msg.progressChan),
			waitForProvisionResult(msg.resultChan),
		)

	case provisionStatusMsg:
		m.provisionStatus = string(msg)
		// Continue waiting for more status updates using stored channel
		if m.provisionChannels != nil {
			return m, waitForProvisionStatus(m.provisionChannels.progressChan)
		}
		return m, nil

	case provisionResultMsg:
		m.provisionChannels = nil // clear channels
		if msg.err != nil {
			m.err = fmt.Errorf("provisioning failed: %w", msg.err)
			m.state = StateError
			return m, nil
		}
		// Success! Transition to search
		m.state = StateSearch
		m.searchInput.Focus()
		return m, textinput.Blink

	case geocodeMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("geocoding failed: %w", msg.err)
			m.state = StateError
			return m, nil
		}
		m.location = msg.location
		// Find nearby zones and nearest tide station
		return m, tea.Batch(
			findNearbyZones(msg.location.Latitude, msg.location.Longitude),
			findNearestTideStation(msg.location.Latitude, msg.location.Longitude),
		)

	case tideStationFoundMsg:
		if msg.err == nil && len(msg.stations) > 0 {
			m.tideStations = msg.stations
			m.tideStation = &msg.stations[0] // Auto-select closest
			// Fetch tide data for this station
			m.loadingTides = true
			return m, fetchTideData(m.tideClient, m.tideStation.ID)
		} else if msg.err != nil {
			// Log error but don't stop app? 
			// For now, if tide lookup fails, we just don't have tide data.
			// Ideally we show a message in the Tide Pane.
			m.tideStation = nil
			m.tideStations = nil
		}
		return m, nil

	case tideDataFetchedMsg:
		m.loadingTides = false
		if msg.err != nil {
			// Handle error, maybe just log or show in UI
		} else {
			m.tides = msg.tides
			m.tideConditions = msg.conditions
			
			// Update chart data
			if m.tides != nil {
				m.tideChart.Clear() // Clear old data first
				
				for _, event := range m.tides.Events {
					m.tideChart.Push(timeserieslinechart.TimePoint{
						Time:  event.Time,
						Value: event.Height,
					})
				}
				m.tideChart.DrawBraille() // Draw chart with new data
			}
		}
		return m, nil

	case zonesFoundMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("finding zones failed: %w", msg.err)
			m.state = StateError
			return m, nil
		}
		if len(msg.zones) == 0 {
			m.err = fmt.Errorf("no marine zones found near '%s'", m.searchQuery)
			m.state = StateError
			return m, nil
		}
		m.zones = msg.zones
		m.zoneList = createZoneList(msg.zones, m.width-4, m.height-10)
		m.state = StateZoneList
		return m, nil

	case zoneWeatherFetchedMsg:
		m.loadingWeather = false
		if msg.err != nil {
			// Keep existing data if fetch failed
		} else {
			m.weather = msg.conditions
			m.forecast = msg.forecast
		}
		// Transition to display if not already there
		if m.state != StateDisplay {
			m.state = StateDisplay
		}
		return m, nil

	case zoneAlertsFetchedMsg:
		m.loadingAlerts = false
		if msg.err != nil {
			// Keep existing data if fetch failed
		} else {
			m.alerts = msg.alerts
		}
		// Transition to display if not already there
		if m.state != StateDisplay {
			m.state = StateDisplay
		}
		return m, nil

	case directLoadStationMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("failed to load station '%s': %w", m.initialStationCode, msg.err)
			m.state = StateError
			return m, nil
		}
		m.selectedZone = msg.zone
		m.state = StateLoading
		m.loadingWeather = true
		m.loadingAlerts = true
		return m, tea.Batch(
			fetchZoneWeather(m.weatherClient, m.selectedZone.Code),
			fetchZoneAlerts(m.alertClient, m.selectedZone.Code),
		)
	}

	// Handle keyboard input
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Global keys
		if keyMsg.String() == "ctrl+c" || keyMsg.String() == "q" {
			return m, tea.Quit
		}

		// State-specific handling
		switch m.state {
		case StateSearch:
			return m.handleSearchInput(keyMsg)

		case StateZoneList:
			return m.handleZoneList(msg)

		case StateDisplay:
			// 's' to return to search
			if keyMsg.String() == "s" {
				m.state = StateSearch
				m.searchInput.SetValue("")
				m.searchInput.Focus()
				// Clear all data
				m.selectedZone = nil
				m.weather = nil
				m.forecast = nil
				m.alerts = nil
				m.location = nil
				m.zones = nil
				return m, textinput.Blink
			}
			// Tab to switch panes
			if keyMsg.Type == tea.KeyTab {
				if m.activePane == PaneWeather {
					m.activePane = PaneTides
				} else {
					m.activePane = PaneWeather
				}
				return m, nil
			}
			// Shift+Tab (same logic for 2 tabs)
			if keyMsg.Type == tea.KeyShiftTab {
				if m.activePane == PaneWeather {
					m.activePane = PaneTides
				} else {
					m.activePane = PaneWeather
				}
				return m, nil
			}
			return m, nil

		case StateError:
			// Any key returns to search (except quit keys)
			m.state = StateSearch
			m.err = nil
			m.searchInput.Focus()
			return m, textinput.Blink
		}
	}

	// Update appropriate component based on state
	switch m.state {
	case StateProvisioning:
		var cmdSpinner tea.Cmd
		m.spinner, cmdSpinner = m.spinner.Update(msg)
		
		// Handle provisioning specific messages here to access local scope if needed, 
		// but we can also handle them in the main switch.
		// Let's handle the channel persistence by using a modified message type.
		// For now, I need to update `zone_messages.go` to include the channel in the msg.
		// I will do that in a separate step before this one compiles?
		// No, I should do it now or rely on the `Update` logic having access to a stored channel in `Model`.
		// storing `provisionChannels` in Model is cleaner than passing them around in messages.
		
		return m, cmdSpinner
		
	case StateSearch:
		m.searchInput, cmd = m.searchInput.Update(msg)
	case StateZoneList:
		m.zoneList, cmd = m.zoneList.Update(msg)
	}

	return m, cmd
}

// handleSearchInput handles keyboard input in search state
func (m Model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Clear error when typing
	if m.err != nil && msg.Type != tea.KeyEnter {
		m.err = nil
	}

	// Handle Enter key
	if msg.Type == tea.KeyEnter {
		query := m.searchInput.Value()
		if query == "" {
			return m, nil
		}
		m.searchQuery = query
		m.err = nil
		m.state = StateLoading
		// Start geocoding
		return m, geocodeLocation(m.geocoder, query)
	}

	// Update text input
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// handleZoneList handles keyboard input in zone list state
func (m Model) handleZoneList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Check for Enter key to select zone
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter {
			// Get selected zone
			if item, ok := m.zoneList.SelectedItem().(zoneItem); ok {
				m.selectedZone = &item.zone
				m.state = StateLoading
				m.loadingWeather = true
				m.loadingAlerts = true
				// Clear old data
				m.weather = nil
				m.forecast = nil
				m.alerts = nil
				// Fetch data for this zone
				return m, tea.Batch(
					fetchZoneWeather(m.weatherClient, m.selectedZone.Code),
					fetchZoneAlerts(m.alertClient, m.selectedZone.Code),
				)
			}
		}
		// 's' or Esc to go back to search
		if keyMsg.String() == "s" || keyMsg.Type == tea.KeyEsc {
			m.state = StateSearch
			m.searchInput.Focus()
			return m, textinput.Blink
		}
	}

	// Update list
	m.zoneList, cmd = m.zoneList.Update(msg)

	return m, cmd
}


// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// 1. Render Background Layer
	// This is either the weather display or a placeholder if no zone is selected
	var background string
	if m.selectedZone != nil {
		background = m.renderWeatherView()
	} else {
		background = m.renderEmptyState()
	}

	// 2. Render Modal Layer (if applicable)
	var modalContent string
	showModal := false

	switch m.state {
	case StateSearch:
		modalContent = m.viewSearch()
		showModal = true
	case StateZoneList:
		modalContent = m.viewZoneList()
		showModal = true
	case StateLoading:
		modalContent = m.viewLoading()
		showModal = true
	case StateProvisioning:
		modalContent = m.viewProvisioning()
		showModal = true
	case StateError:
		modalContent = m.viewError()
		showModal = true
	}

	// 3. Composite layers
	if showModal {
		
		modal := modalStyle.Render(modalContent)
		
		// Use lipgloss.Place to center the modal over the background
		// We place the modal in a box the size of the window
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			modal,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(colorMuted),
		)
	}

	return background
}

// renderEmptyState renders a placeholder when no station is selected
func (m Model) renderEmptyState() string {
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			titleStyle.Render("⚓ Marine Terminal"),
			mutedStyle.Render("Press 'S' to search for a station"),
		),
	)
}

// viewProvisioning renders the initial setup screen
func (m Model) viewProvisioning() string {
	title := titleStyle.Render("⚓ Setup")
	
	sp := m.spinner.View()
	status := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(m.provisionStatus)
		
	info := helpStyle.Render("Downloading marine zones...")

	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		fmt.Sprintf("%s %s", sp, status),
		"",
		info,
	)
}

// viewError renders the error view
func (m Model) viewError() string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B6B")).
		Bold(true).
		Render("✗ Error")

	var errorMsg string
	if m.err != nil {
		errorMsg = m.err.Error()
	} else {
		errorMsg = "An unknown error occurred"
	}

	help := helpStyle.Render("Esc: Back • Q: Quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		errorMsg,
		"",
		help,
	)
}

// viewSearch renders the search modal content
func (m Model) viewSearch() string {
	// Title
	title := titleStyle.Render("Search Station")
	subtitle := mutedStyle.Render("Enter Zipcode or City, State")

	// Search box with border
	searchBox := m.searchInput.View()

	// Error message if present
	var errorMsg string
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)
		errorMsg = errorStyle.Render("✗ " + m.err.Error())
	}

	// Assemble the view
	var sections []string
	sections = append(sections, title)
	sections = append(sections, subtitle)
	sections = append(sections, "")
	sections = append(sections, searchBox)

	if m.err != nil {
		sections = append(sections, "")
		sections = append(sections, errorMsg)
	}

	// Examples (compact)
	examples := mutedStyle.Render("e.g. 02633, Chatham MA")
	sections = append(sections, "", examples)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// viewZoneList renders the marine zone selection list for modal
func (m Model) viewZoneList() string {
	title := titleStyle.Render("Select Zone")
	//subtitle := mutedStyle.Render(fmt.Sprintf("Found %d zones", len(m.zones)))

	return lipgloss.JoinVertical(lipgloss.Left, 
		title, 
		//subtitle, 
		"", 
		m.zoneList.View(),
	)
}

// viewLoading renders the loading modal
func (m Model) viewLoading() string {
	sp := m.spinner.View()
	label := "Loading..."

	return fmt.Sprintf("%s %s", sp, label)
}

// renderWeatherView renders the main display with tabs
func (m Model) renderWeatherView() string {
	if m.selectedZone == nil {
		return "No zone selected"
	}

	var sections []string

	// Header with nice styling
	headerStyle := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Padding(0, 1).
		MarginBottom(1)
	header := headerStyle.Render(fmt.Sprintf("⚓ %s - %s", m.selectedZone.Code, m.selectedZone.Name))
	sections = append(sections, header)

	// Location info
	if m.location != nil {
		locationInfo := mutedStyle.Render(fmt.Sprintf("📍 %s (%.1f mi away)", m.searchQuery, m.selectedZone.Distance))
		sections = append(sections, locationInfo, "")
	}

	// Tab Bar
	var tabs []string
	
	weatherTab := "Weather"
	if m.activePane == PaneWeather {
		weatherTab = activeTabStyle.Render(weatherTab)
	} else {
		weatherTab = tabStyle.Render(weatherTab)
	}
	tabs = append(tabs, weatherTab)

	tidesTab := "Tides"
	if m.activePane == PaneTides {
		tidesTab = activeTabStyle.Render(tidesTab)
	} else {
		tidesTab = tabStyle.Render(tidesTab)
	}
	tabs = append(tabs, tidesTab)

	// Remove Alerts tab from tab bar
	
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	sections = append(sections, tabBar)
	sections = append(sections, "") // Spacer

	// Determine available width for boxes
	// Full width - 2 (left/right margin) - 2 (border) - 4 (padding)
	boxWidth := m.width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}
	
	// Create box style with dynamic width
	boxStyle := sectionBoxStyle.Copy().Width(boxWidth)

	// Content based on active pane
	var content string
	switch m.activePane {
	case PaneWeather:
		// Weather/Forecast section
		weatherSection := lipgloss.JoinVertical(lipgloss.Left,
			boxHeaderStyle.Render("⛅ MARINE FORECAST"),
			m.renderWeatherSimple(),
		)
		
		// Alerts section (now part of Weather pane)
		alertSection := lipgloss.JoinVertical(lipgloss.Left,
			boxHeaderStyle.Render("⚠️  MARINE ALERTS"),
			m.renderAlertSimple(),
		)
		
		// Join them with some spacing
		content = lipgloss.JoinVertical(lipgloss.Left,
			weatherSection,
			"", // Spacer
			alertSection,
		)
		
	case PaneTides:
		var tideInfo string
		if m.tideStation != nil {
			tideInfo = fmt.Sprintf("Station: %s (%s)\n", m.tideStation.Name, m.tideStation.ID)
			
			if m.loadingTides {
				tideInfo += "\n" + m.spinner.View() + " Loading tide predictions..."
			} else {
				if m.tideConditions != nil {
					tideInfo += fmt.Sprintf("Air Temp: %.1f°F  Pressure: %.1f mb\n", 
						m.tideConditions.Temperature, 
						m.tideConditions.Pressure)
				}
				
				if m.tides != nil {
					// Render chart
					tideInfo += "\n" + m.tideChart.View()
					
					// Render list below chart
					tideInfo += "\n\nUpcoming Tides:"
					for i, event := range m.tides.Events {
						if i >= 6 { break } // Limit to 6
						tideInfo += fmt.Sprintf("\n  %s  %-4s  %.1f ft", 
							event.Time.Format("Jan 2, 3:04 PM"),
							event.Type,
							event.Height,
						)
					}
				} else {
					tideInfo += "\nNo tide predictions available."
				}
			}
		} else {
			tideInfo = "No nearby tide station found."
		}
		
		content = lipgloss.JoinVertical(lipgloss.Left,
			boxHeaderStyle.Render("🌊 TIDES"),
			tideInfo,
		)
	}

	sections = append(sections, boxStyle.Render(content))

	// Help text
	help := helpStyle.Render("S: New search • Tab: Switch tab • Q: Quit")

	sections = append(sections, "", help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderWeatherSimple renders the current weather conditions
func (m Model) renderWeatherSimple() string {
	if m.loadingWeather {
		return fmt.Sprintf("%s Fetching marine forecast...", m.spinner.View())
	}

	if m.weather == nil {
		return "No marine weather data available."
	}

	return formatWeather(m.weather, m.forecast)
}

// renderAlertSimple renders active marine alerts
func (m Model) renderAlertSimple() string {
	if m.loadingAlerts {
		return fmt.Sprintf("%s Fetching marine alerts...", m.spinner.View())
	}

	if m.alerts == nil || len(m.alerts.Alerts) == 0 {
		return "No active marine alerts."
	}

	return formatAlerts(m.alerts)
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

// formatWeather renders the current weather conditions
func formatWeather(current *models.MarineConditions, forecast *models.ThreeDayForecast) string {
	if current == nil && forecast == nil {
		return mutedStyle.Render("No weather data available")
	}

	var lines []string

	// Current conditions
	if current != nil && forecast != nil && len(forecast.Periods) > 0 {
		periodStyle := lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)
		lines = append(lines, periodStyle.Render(forecast.Periods[0].PeriodName))

		if current.Wind.Direction != "" {
			windLabel := labelStyle.Render("Wind: ")
			windValue := valueStyle.Render(formatWind(current.Wind))
			lines = append(lines, windLabel+windValue)
		}

		if current.Seas.HeightMin > 0 || current.Seas.HeightMax > 0 {
			seasLabel := labelStyle.Render("Seas: ")
			seasValue := valueStyle.Render(formatSeas(current.Seas))
			lines = append(lines, seasLabel+seasValue)
		}

		if len(current.Seas.Components) > 0 {
			for _, wave := range current.Seas.Components {
				waveText := fmt.Sprintf("  %s %.0f ft at %d sec", wave.Direction, wave.Height, wave.Period)
				lines = append(lines, mutedStyle.Render(waveText))
			}
		}
	}

	// Forecast
	if forecast != nil && len(forecast.Periods) > 1 {
		lines = append(lines, "", labelStyle.Render("📅 3-Day Forecast:"))

		maxPeriods := 6
		if len(forecast.Periods)-1 < maxPeriods {
			maxPeriods = len(forecast.Periods) - 1
		}

		for i := 1; i <= maxPeriods; i++ {
			period := forecast.Periods[i]
			periodName := valueStyle.Render(period.PeriodName + ":")
			windSeas := mutedStyle.Render(fmt.Sprintf("%s, Seas %s",
				formatWind(period.Wind),
				formatSeas(period.Seas)))
			lines = append(lines, fmt.Sprintf("  %s %s", periodName, windSeas))
		}
	}

	return strings.Join(lines, "\n")
}

// formatAlerts renders active marine alerts
func formatAlerts(alerts *models.AlertData) string {
	if alerts == nil {
		return mutedStyle.Render("No alert data available")
	}

	activeAlerts := make([]models.Alert, 0)
	for _, alert := range alerts.Alerts {
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