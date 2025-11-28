package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ngmaloney/marine-terminal/internal/database"
	"github.com/ngmaloney/marine-terminal/internal/geocoding"
	"github.com/ngmaloney/marine-terminal/internal/models"
	"github.com/ngmaloney/marine-terminal/internal/noaa"
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
	PaneAlerts
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

	// API clients
	weatherClient noaa.WeatherClient
	alertClient   noaa.AlertClient

	// Data
	weather  *models.MarineConditions
	forecast *models.ThreeDayForecast
	alerts   *models.AlertData

	// Loading states
	loadingWeather bool
	loadingAlerts  bool

	// Provisioning
	spinner           spinner.Model
	provisionStatus   string
	provisionChannels *provisioningStartedMsg
}

// NewModel creates a new application model
func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter zipcode or city, state (e.g. 02633 or Chatham, MA)..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 60

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		state:         StateSearch, // Will be checked in Init
		activePane:    PaneWeather,
		searchInput:   ti,
		geocoder:      geocoding.NewGeocoder(),
		weatherClient: noaa.NewWeatherClient(),
		alertClient:   noaa.NewAlertClient(),
		spinner:       s,
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
		// Find nearby zones
		return m, findNearbyZones(msg.location.Latitude, msg.location.Longitude)

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
		// Transition to display if all done
		if !m.loadingWeather && !m.loadingAlerts {
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
		// Transition to display if all done
		if !m.loadingWeather && !m.loadingAlerts {
			m.state = StateDisplay
		}
		return m, nil
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
					m.activePane = PaneAlerts
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

	// Check if we transitioned to display after fetching
	if m.state == StateLoading && !m.loadingWeather && !m.loadingAlerts {
		m.state = StateDisplay
	}

	return m, cmd
}


// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.state {
	case StateProvisioning:
		return m.viewProvisioning()
	case StateSearch:
		return m.viewSearch()
	case StateZoneList:
		return m.viewZoneList()
	case StateLoading:
		return m.viewLoading()
	case StateDisplay:
		return m.viewDisplay()
	case StateError:
		return m.viewError()
	}

	return ""
}

// viewProvisioning renders the initial setup screen
func (m Model) viewProvisioning() string {
	title := titleStyle.Render("⚓ Marine Terminal Setup")

	sp := m.spinner.View()
	status := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(m.provisionStatus)

	info := helpStyle.Render("One-time setup: downloading marine zones database...")

	return lipgloss.JoinVertical(
		lipgloss.Center,
		"",
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

	help := helpStyle.Render("Press any key to return to search • Q: Quit")

	var sections []string
	sections = append(sections, title)
	sections = append(sections, "")
	sections = append(sections, errorMsg)
	sections = append(sections, "")
	sections = append(sections, help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// viewSearch renders the search view
func (m Model) viewSearch() string {
	// Title
	title := titleStyle.Render("⚓ Marine Terminal")
	subtitle := mutedStyle.Render("NOAA Marine Weather & Alerts")

	// Search box with border
	searchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(64).
		Render(m.searchInput.View())

	// Error message if present
	var errorMsg string
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true).
			Padding(0, 2)
		errorMsg = errorStyle.Render("✗ " + m.err.Error())
	}

	// Help text
	help := helpStyle.Render("Press Enter to search • Ctrl+C to quit")

	// Examples
	examples := mutedStyle.Render("Examples: 02633 | Chatham, MA | Boston, MA | Seattle, WA")

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

	sections = append(sections, "")
	sections = append(sections, examples)
	sections = append(sections, "")
	sections = append(sections, help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// viewZoneList renders the marine zone selection list
func (m Model) viewZoneList() string {
	title := titleStyle.Render("⚓ Marine Zones")
	subtitle := mutedStyle.Render(fmt.Sprintf("Found %d zones near %s", len(m.zones), m.searchQuery))

	help := helpStyle.Render("↑/↓: Navigate • Enter: Select • S/Esc: Back to search • Q: Quit")

	var sections []string
	sections = append(sections, title)
	sections = append(sections, subtitle)
	sections = append(sections, "")
	sections = append(sections, m.zoneList.View())
	sections = append(sections, "")
	sections = append(sections, help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// viewLoading renders the loading view
func (m Model) viewLoading() string {
	s := "Loading marine data"
	if m.selectedZone != nil {
		s += fmt.Sprintf(" for %s", m.selectedZone.Code)
	}
	s += "...\n\n"

	if m.loadingWeather {
		s += "⏳ Fetching weather forecast\n"
	} else {
		s += "✓ Weather forecast loaded\n"
	}

	if m.loadingAlerts {
		s += "⏳ Fetching marine alerts\n"
	} else {
		s += "✓ Alerts loaded\n"
	}

	// Auto-transition to display when done
	if !m.loadingWeather && !m.loadingAlerts {
		s += "\n✓ Ready!"
	}

	return s
}

// viewDisplay renders the main display - simple vertical layout
func (m Model) viewDisplay() string {
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

	// Weather/Forecast section
	sections = append(sections,
		sectionHeaderStyle.Render("⛅ MARINE FORECAST"),
		m.renderWeatherSimple(),
	)

	// Alerts section
	sections = append(sections,
		sectionHeaderStyle.Render("⚠️  MARINE ALERTS"),
		m.renderAlertSimple(),
	)

	// Help text
	help := helpStyle.Render("S: New search • Tab: Switch panes • Q: Quit")

	sections = append(sections, "", help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
