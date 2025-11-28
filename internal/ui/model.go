package ui

import (
	"fmt"
	"strings"

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
	"github.com/ngmaloney/marine-terminal/internal/ports"
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
	StateSavedPorts                   // List saved ports
	StateSavePrompt                   // Prompt for saving a port
	StateConfirmDelete                // Prompt for confirming deletion of a port
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

	// Services
	portService *ports.Service

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

	// Ports
	savedPorts []models.Port
	portList   list.Model
	saveInput  textinput.Model
	saving     bool
	portToDelete *models.Port // New: for confirmation before deleting

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
	initialLocation    string
	initialPortName    string
}

// NewModel creates a new application model
func NewModel(initialStationCode, initialLocation, initialPortName string) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter zipcode or city, state (e.g. 02633 or Chatham, MA)..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 60

	si := textinput.New()
	si.Placeholder = "Enter a name for this port (e.g. Stage Harbor)"
	si.CharLimit = 50
	si.Width = 60

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	tc := timeserieslinechart.New(80, 15) // Initial size, will be resized on first WindowSizeMsg
	
	return Model{
		state:         StateLoading, // Start in loading to check for saved ports
		activePane:    PaneWeather,
		searchInput:   ti,
		saveInput:     si,
		geocoder:      geocoding.NewGeocoder(),
		weatherClient: noaa.NewWeatherClient(),
		alertClient:   noaa.NewAlertClient(),
		tideClient:    noaa.NewTideClient(),
		portService:   ports.NewService(),
		spinner:       s,
		tideChart:     tc,
		initialStationCode: initialStationCode,
		initialLocation:    initialLocation,
		initialPortName:    initialPortName,
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	dbPath := database.DBPath()
	
	// Check if we need to provision the database (marine zones or zipcodes)
	zonesNeeded, err := zonelookup.NeedsProvisioning(dbPath)
	if err == nil {
		zipNeeded, err := geocoding.NeedsProvisioning(dbPath)
		if err == nil && (zonesNeeded || zipNeeded) {
			return tea.Batch(m.spinner.Tick, initiateProvisioning())
		}
	}

	// 1. Load by Port Name
	if m.initialPortName != "" {
		return tea.Batch(m.spinner.Tick, fetchPortByName(m.portService, m.initialPortName))
	}

	// 2. Load by Station + Location
	if m.initialStationCode != "" && m.initialLocation != "" {
		m.searchQuery = m.initialLocation
		return tea.Batch(m.spinner.Tick, geocodeLocation(m.geocoder, m.initialLocation))
	}

	// 3. Default: Fetch saved ports
	return tea.Batch(m.spinner.Tick, fetchSavedPorts(m.portService))
}

// loadPort sets up the model to display a specific port
func (m Model) loadPort(p models.Port) (Model, tea.Cmd) {
	if p.Zipcode != "" {
		m.searchQuery = p.Zipcode
	} else {
		m.searchQuery = fmt.Sprintf("%s, %s", p.City, p.State)
	}
	m.selectedZone = &zonelookup.ZoneInfo{
		Code: p.MarineZoneID,
		Name: p.Name, 
	}
	m.location = &geocoding.Location{
		Latitude:  p.Latitude,
		Longitude: p.Longitude,
		Name:      m.searchQuery,
	}
	m.state = StateLoading
	m.loadingWeather = true
	m.loadingAlerts = true
	return m, tea.Batch(
		fetchZoneWeather(m.weatherClient, m.selectedZone.Code),
		fetchZoneAlerts(m.alertClient, m.selectedZone.Code),
		findNearestTideStation(m.location.Latitude, m.location.Longitude),
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle window size
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		// Update lists size if they exist
		if m.state == StateZoneList {
			m.zoneList.SetSize(msg.Width-4, msg.Height-10)
		}
		if m.state == StateSavedPorts {
			m.portList.SetSize(msg.Width-4, msg.Height-10)
		}
		// Update tide chart size based on terminal width
		chartWidth := msg.Width - 8 // Leave some padding
		if chartWidth < 40 {
			chartWidth = 40 // Minimum width
		}
		m.tideChart = timeserieslinechart.New(chartWidth, 15)
		// If we have tide data, redraw the chart
		if m.tides != nil {
			m.tideChart.Clear()
			for _, event := range m.tides.Events {
				m.tideChart.Push(timeserieslinechart.TimePoint{
					Time:  event.Time,
					Value: event.Height,
				})
			}
			m.tideChart.DrawBraille()
		}
		return m, nil
	}

	// Handle custom messages
	switch msg := msg.(type) {
	case errMsg:
		m.err = msg.err
		m.state = StateError
		return m, nil

	case portsFetchedMsg:
		if msg.err != nil {
			// If error fetching ports, default to search
			m.state = StateSearch
			m.searchInput.Focus()
			return m, nil
		}
		m.savedPorts = msg.ports
		
		// If ports exist, populate the list
		if len(m.savedPorts) > 0 {
			m.portList = createPortList(m.savedPorts, m.width-4, m.height-10)
			// AUTO-LOAD: If we have ports, load the first one by default
			return m.loadPort(m.savedPorts[0])
		}
		
		// No ports, go to search (new port setup)
		m.state = StateSearch
		m.searchInput.Focus()
		return m, nil

	case portFetchedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = StateError
			return m, nil
		}
		return m.loadPort(*msg.port)

	case portSavedMsg:
		m.saving = false
		if msg.err != nil {
			m.err = msg.err
			m.state = StateError
			return m, nil
		}
		
		// Update local list
		found := false
		for i, p := range m.savedPorts {
			if p.Name == msg.port.Name {
				m.savedPorts[i] = *msg.port
				found = true
				break
			}
		}
		if !found {
			m.savedPorts = append(m.savedPorts, *msg.port)
		}
		
		// Update UI list
		m.portList = createPortList(m.savedPorts, m.width-4, m.height-10)
		
		return m.loadPort(*msg.port)

	case portDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = StateError
			return m, nil
		}
		// Remove the deleted port from the list
		var updatedPorts []models.Port
		for _, p := range m.savedPorts {
			if p.Name != msg.name {
				updatedPorts = append(updatedPorts, p)
			}
		}
		m.savedPorts = updatedPorts
		m.portList = createPortList(m.savedPorts, m.width-4, m.height-10)
		m.state = StateSavedPorts // Return to saved ports list
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
		// Success! Check saved ports again
		return m, fetchSavedPorts(m.portService)

	case geocodeMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("geocoding failed: %w", msg.err)
			m.state = StateError
			return m, nil
		}
		m.location = msg.location
		
		// If we are direct loading with station code
		if m.initialStationCode != "" {
			// Skip zone finding, assume station code is valid
			m.selectedZone = &zonelookup.ZoneInfo{
				Code: m.initialStationCode,
				Name: "Direct Loaded",
			}
			m.state = StateLoading
			m.loadingWeather = true
			m.loadingAlerts = true
			return m, tea.Batch(
				fetchZoneWeather(m.weatherClient, m.selectedZone.Code),
				fetchZoneAlerts(m.alertClient, m.selectedZone.Code),
				findNearestTideStation(m.location.Latitude, m.location.Longitude),
			)
		}

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
		if m.state != StateDisplay {
			m.state = StateDisplay
		}
		return m, nil

	}

	// Handle keyboard input
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Global keys
		if keyMsg.String() == "ctrl+c" || keyMsg.String() == "q" {
			// Allow quitting unless in input fields where 'q' might be text
			if m.state != StateSearch && m.state != StateSavePrompt {
				return m, tea.Quit
			}
			// In inputs, ctrl+c quits
			if keyMsg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		}

		// State-specific handling
		switch m.state {
		case StateSearch:
			return m.handleSearchInput(keyMsg)

		case StateSavedPorts:
			return m.handleSavedPorts(msg)

		case StateSavePrompt:
			return m.handleSavePrompt(keyMsg)

		case StateConfirmDelete:
			return m.handleConfirmDelete(keyMsg)

		case StateZoneList:
			return m.handleZoneList(msg)

		case StateDisplay:
			// 'e' to edit/change port
			if keyMsg.String() == "e" {
				m.state = StateSavedPorts
				return m, nil
			}
			// Tab to switch panes
			if keyMsg.Type == tea.KeyTab || keyMsg.Type == tea.KeyShiftTab {
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
		return m, cmdSpinner
		
	case StateSearch:
		m.searchInput, cmd = m.searchInput.Update(msg)
	case StateSavePrompt:
		m.saveInput, cmd = m.saveInput.Update(msg)
	// StateZoneList is handled by handleZoneList() above, don't update twice
	case StateSavedPorts:
		m.portList, cmd = m.portList.Update(msg)
	}

	return m, cmd
}

func (m Model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.err != nil && msg.Type != tea.KeyEnter {
		m.err = nil
	}
	if msg.Type == tea.KeyEnter {
		query := m.searchInput.Value()
		if query == "" {
			return m, nil
		}
		m.searchQuery = query
		m.err = nil
		m.state = StateLoading
		return m, geocodeLocation(m.geocoder, query)
	}
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m Model) handleSavePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if msg.Type == tea.KeyEsc {
		m.state = StateDisplay
		return m, nil
	}
	if msg.Type == tea.KeyEnter {
		name := m.saveInput.Value()
		if name == "" {
			return m, nil
		}
		m.saving = true
		return m, savePort(m.portService, name, m.searchQuery, m.selectedZone.Code)
	}
	m.saveInput, cmd = m.saveInput.Update(msg)
	return m, cmd
}

func (m Model) handleSavedPorts(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter {
			if item, ok := m.portList.SelectedItem().(portItem); ok {
				return m.loadPort(item.port)
			}
		}
		if keyMsg.String() == "n" {
			m.state = StateSearch
			m.searchInput.Focus()
			return m, textinput.Blink
		}
		// Handle escape key - return to weather pane if we have a selected zone
		if keyMsg.Type == tea.KeyEsc {
			if m.selectedZone != nil {
				m.state = StateDisplay
				return m, nil
			}
			// If no zone selected, stay in saved ports
			return m, nil
		}
		// New: handle delete key
		if keyMsg.String() == "d" {
			if item, ok := m.portList.SelectedItem().(portItem); ok {
				m.portToDelete = &item.port
				m.state = StateConfirmDelete
				return m, nil
			}
		}
	}
	m.portList, cmd = m.portList.Update(msg)
	return m, cmd
}

func (m Model) handleConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "y" || msg.String() == "Y" {
		if m.portToDelete != nil {
			return m, deletePort(m.portService, m.portToDelete.Name)
		}
	} else if msg.String() == "n" || msg.String() == "N" || msg.Type == tea.KeyEsc {
		// Cancel deletion
		m.portToDelete = nil
		m.state = StateSavedPorts
		return m, nil
	}
	return m, nil
}

func (m Model) handleZoneList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter {
			if item, ok := m.zoneList.SelectedItem().(zoneItem); ok {
				m.selectedZone = &item.zone
				// Transition to save prompt to define the port
				m.state = StateSavePrompt
				// Default name to location (city/state) or search query
				defaultName := m.searchQuery
				if m.location != nil && m.location.Name != "" {
					defaultName = m.location.Name
				}
				m.saveInput.SetValue(defaultName)
				m.saveInput.Focus()
				return m, nil
			}
		}
		if keyMsg.String() == "s" || keyMsg.Type == tea.KeyEsc {
			m.state = StateSearch
			m.searchInput.Focus()
			return m, textinput.Blink
		}
	}
	m.zoneList, cmd = m.zoneList.Update(msg)
	return m, cmd
}

// View and render methods
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	var background string
	if m.selectedZone != nil {
		background = m.renderWeatherView()
	} else {
		background = m.renderEmptyState()
	}
	var modalContent string
	showModal := false
	switch m.state {
	case StateSearch:
		modalContent = m.viewSearch()
		showModal = true
	case StateZoneList:
		modalContent = m.viewZoneList()
		showModal = true
	case StateSavedPorts:
		modalContent = m.viewSavedPorts()
		showModal = true
	case StateSavePrompt:
		modalContent = m.viewSavePrompt()
		showModal = true
	case StateConfirmDelete:
		modalContent = m.viewConfirmDelete()
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
	if showModal {
		modal := modalStyle.Render(modalContent)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(colorMuted))
	}
	return background
}

func (m Model) renderEmptyState() string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, lipgloss.JoinVertical(lipgloss.Center, titleStyle.Render("⚓ Marine Terminal"), mutedStyle.Render("Press 'E' to view ports")))
}

func (m Model) viewProvisioning() string {
	sp := m.spinner.View()
	status := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(m.provisionStatus)
	return lipgloss.JoinVertical(lipgloss.Center, titleStyle.Render("⚓ Setup"), "", fmt.Sprintf("%s %s", sp, status), "", helpStyle.Render("Downloading marine zones..."))
}

func (m Model) viewError() string {
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Bold(true).Render("✗ Error")
	msg := "An unknown error occurred"
	if m.err != nil { msg = m.err.Error() }
	return lipgloss.JoinVertical(lipgloss.Left, title, "", msg, "", helpStyle.Render("Esc: Back • Q: Quit"))
}

func (m Model) viewSearch() string {
	title := titleStyle.Render("New Port Setup")
	subtitle := mutedStyle.Render("Enter Zipcode or City, State")
	sb := m.searchInput.View()
	errorMsg := ""
	if m.err != nil {
		errorMsg = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Bold(true).Render("✗ " + m.err.Error())
	}
	content := []string{title, subtitle, "", sb}
	if errorMsg != "" { content = append(content, "", errorMsg) }
	content = append(content, "", mutedStyle.Render("e.g. 02633, Chatham MA"))
	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func (m Model) viewSavedPorts() string {
	title := titleStyle.Render("Saved Ports")
	help := mutedStyle.Render("Enter: Select • n: New Port • d: Delete Port")
	return lipgloss.JoinVertical(lipgloss.Left, title, "", m.portList.View(), "", help)
}

func (m Model) viewSavePrompt() string {
	title := titleStyle.Render("Save Port")
	subtitle := mutedStyle.Render("Enter a name for this configuration")
	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", m.saveInput.View())
}

func (m Model) viewConfirmDelete() string {
	portName := ""
	if m.portToDelete != nil {
		portName = m.portToDelete.Name
	}

	title := alertDangerStyle.Render("Delete Port")
	prompt := fmt.Sprintf("Are you sure you want to delete '%s'? (y/n)", portName)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", prompt, "", helpStyle.Render("y: Confirm • n/Esc: Cancel"))
}

func (m Model) viewZoneList() string {
	return lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("Select Zone"), "", m.zoneList.View())
}

func (m Model) viewLoading() string {
	return fmt.Sprintf("%s Loading...", m.spinner.View())
}

func (m Model) renderWeatherView() string {
	if m.selectedZone == nil { return "No zone" }
	header := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Padding(0, 1).MarginBottom(1).Render(fmt.Sprintf("⚓ %s - %s", m.selectedZone.Code, m.selectedZone.Name))
	loc := ""
	if m.location != nil {
		loc = mutedStyle.Render(fmt.Sprintf("📍 %s (%.1f mi away)", m.searchQuery, m.selectedZone.Distance))
	}
	
	weatherTab := tabStyle.Render("Weather")
	if m.activePane == PaneWeather { weatherTab = activeTabStyle.Render("Weather") }
	tidesTab := tabStyle.Render("Tides")
	if m.activePane == PaneTides { tidesTab = activeTabStyle.Render("Tides") }
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, weatherTab, tidesTab)
	
	boxWidth := m.width - 4
	if boxWidth < 40 { boxWidth = 40 }
	boxStyle := sectionBoxStyle.Copy().Width(boxWidth)
	
	var content string
	if m.activePane == PaneWeather {
		content = lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.JoinVertical(lipgloss.Left, boxHeaderStyle.Render("⛅ MARINE FORECAST"), m.renderWeatherSimple()),
			"",
			lipgloss.JoinVertical(lipgloss.Left, boxHeaderStyle.Render("⚠️  MARINE ALERTS"), m.renderAlertSimple()),
		)
	} else {
		tideInfo := "No nearby tide station found."
		if m.tideStation != nil {
			tideInfo = fmt.Sprintf("Station: %s (%s)\n", m.tideStation.Name, m.tideStation.ID)
			if m.loadingTides {
				tideInfo += "\n" + m.spinner.View() + " Loading tide predictions..."
			} else {
				if m.tideConditions != nil {
					tideInfo += fmt.Sprintf("Air Temp: %.1f°F  Pressure: %.1f mb\n", m.tideConditions.Temperature, m.tideConditions.Pressure)
				}
				if m.tides != nil {
					tideInfo += "\nUpcoming Tides:"
					for i, event := range m.tides.Events {
						if i >= 6 { break }
						tideInfo += fmt.Sprintf("\n  %s  %-4s  %.1f ft", event.Time.Format("Jan 2, 3:04 PM"), event.Type, event.Height)
					}
					tideInfo += "\n\n" + m.tideChart.View()
				} else { tideInfo += "\nNo tide predictions available." }
			}
		}
		content = lipgloss.JoinVertical(lipgloss.Left, boxHeaderStyle.Render("🌊 TIDES"), tideInfo)
	}
	
	return lipgloss.JoinVertical(lipgloss.Left, header, loc, "", tabBar, "", boxStyle.Render(content), "", helpStyle.Render("e: Edit Port • Tab: Switch tab • q: Quit"))
}

func (m Model) renderWeatherSimple() string {
	if m.loadingWeather { return fmt.Sprintf("%s Fetching marine forecast...", m.spinner.View()) }
	if m.weather == nil { return "No marine weather data available." }
	return formatWeather(m.weather, m.forecast)
}

func (m Model) renderAlertSimple() string {
	if m.loadingAlerts { return fmt.Sprintf("%s Fetching marine alerts...", m.spinner.View()) }
	if m.alerts == nil || len(m.alerts.Alerts) == 0 { return "No active marine alerts." }
	return formatAlerts(m.alerts)
}

func formatWind(wind models.WindData) string {
	if wind.SpeedMin == wind.SpeedMax {
		if wind.HasGust { return fmt.Sprintf("%s %0.f kt, gusts %0.f kt", wind.Direction, wind.SpeedMin, wind.GustSpeed) }
		return fmt.Sprintf("%s %.0f kt", wind.Direction, wind.SpeedMin)
	}
	if wind.HasGust { return fmt.Sprintf("%s %.0f-%.0f kt, gusts %.0f kt", wind.Direction, wind.SpeedMin, wind.SpeedMax, wind.GustSpeed) }
	return fmt.Sprintf("%s %.0f-%.0f kt", wind.Direction, wind.SpeedMin, wind.SpeedMax)
}

func formatSeas(seas models.SeaState) string {
	if seas.HeightMin == seas.HeightMax { return fmt.Sprintf("%.0f ft", seas.HeightMin) }
	return fmt.Sprintf("%.0f-%.0f ft", seas.HeightMin, seas.HeightMax)
}

func formatWeather(current *models.MarineConditions, forecast *models.ThreeDayForecast) string {
	if current == nil && forecast == nil { return mutedStyle.Render("No weather data available") }
	var lines []string
	if current != nil && forecast != nil && len(forecast.Periods) > 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(colorSecondary).Bold(true).Render(forecast.Periods[0].PeriodName))
		if current.Wind.Direction != "" { lines = append(lines, labelStyle.Render("Wind: ") + valueStyle.Render(formatWind(current.Wind))) }
		if current.Seas.HeightMin > 0 || current.Seas.HeightMax > 0 { lines = append(lines, labelStyle.Render("Seas: ") + valueStyle.Render(formatSeas(current.Seas))) }
		for _, wave := range current.Seas.Components { lines = append(lines, mutedStyle.Render(fmt.Sprintf("  %s %.0f ft at %d sec", wave.Direction, wave.Height, wave.Period))) }
	}
	if forecast != nil && len(forecast.Periods) > 1 {
		lines = append(lines, "", labelStyle.Render("📅 3-Day Forecast:"))
		max := 6
		if len(forecast.Periods)-1 < max { max = len(forecast.Periods)-1 }
		for i := 1; i <= max; i++ {
			p := forecast.Periods[i]
			lines = append(lines, fmt.Sprintf("  %s %s", valueStyle.Render(p.PeriodName+":"), mutedStyle.Render(fmt.Sprintf("%s, Seas %s", formatWind(p.Wind), formatSeas(p.Seas)))))
		}
	}
	return strings.Join(lines, "\n")
}

func formatAlerts(alerts *models.AlertData) string {
	if alerts == nil { return mutedStyle.Render("No alert data available") }
	activedAlerts := make([]models.Alert, 0)
	for _, a := range alerts.Alerts { if a.IsActive() && a.IsMarine() { activedAlerts = append(activedAlerts, a) } }
	if len(activedAlerts) == 0 { return successStyle.Bold(true).Render("✓ No active marine alerts") }
	var lines []string
	for i, a := range activedAlerts {
		if i > 0 { lines = append(lines, "") }
		lines = append(lines, getAlertStyle(a.Severity).Render(fmt.Sprintf("️%s", a.Event)))
		lines = append(lines, valueStyle.Render(a.Headline))
		lines = append(lines, labelStyle.Render("Expires: ") + mutedStyle.Render(a.Expires.Format("Jan 2, 3:04 PM")))
	}
	return strings.Join(lines, "\n")
}

func getAlertStyle(s models.AlertSeverity) lipgloss.Style {
	switch s {
	case models.SeverityExtreme: return alertExtremeStyle
	case models.SeveritySevere: return alertSevereStyle
	case models.SeverityModerate: return alertModerateStyle
	case models.SeverityMinor: return alertMinorStyle
	default: return valueStyle
	}
}
