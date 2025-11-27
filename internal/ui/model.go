package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ngmaloney/mariner-tui/internal/models"
	"github.com/ngmaloney/mariner-tui/internal/noaa"
	"github.com/ngmaloney/mariner-tui/internal/ports"
)

// AppState represents the current state of the application
type AppState int

const (
	StatePortSearch AppState = iota
	StateLoading
	StateDisplay
	StateError
)

// ActivePane represents which pane is currently focused
type ActivePane int

const (
	PaneWeather ActivePane = iota
	PaneTides
	PaneAlerts
)

// Model represents the application's state
type Model struct {
	state       AppState
	activePane  ActivePane
	width       int
	height      int
	err         error

	// Port information
	currentPort   *models.Port
	searchInput   textinput.Model
	searchResults []models.Port
	portClient    ports.Client

	// API clients
	weatherClient noaa.WeatherClient
	tideClient    noaa.TideClient
	alertClient   noaa.AlertClient

	// Data
	weather  *models.MarineConditions
	forecast *models.ThreeDayForecast
	tides    *models.TideData
	alerts   *models.AlertData

	// Loading states
	loadingWeather bool
	loadingTides   bool
	loadingAlerts  bool
}

// NewModel creates a new application model
func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter city, state, or ZIP code..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	return Model{
		state:         StatePortSearch,
		activePane:    PaneWeather,
		searchInput:   ti,
		portClient:    ports.NewNOAAStationClient(),
		weatherClient: noaa.NewWeatherClient(),
		tideClient:    noaa.NewTideClient(),
		alertClient:   noaa.NewAlertClient(),
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle data fetch messages
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case weatherFetchedMsg:
		m.loadingWeather = false
		if msg.err != nil {
			// Keep existing data if fetch failed
			return m, nil
		}
		m.weather = msg.conditions
		m.forecast = msg.forecast
		return m, nil

	case tidesFetchedMsg:
		m.loadingTides = false
		if msg.err != nil {
			// Keep existing data if fetch failed
			return m, nil
		}
		m.tides = msg.tides
		return m, nil

	case alertsFetchedMsg:
		m.loadingAlerts = false
		if msg.err != nil {
			// Keep existing data if fetch failed
			return m, nil
		}
		m.alerts = msg.alerts
		return m, nil

	case errMsg:
		m.err = msg.err
		m.state = StateError
		return m, nil
	}

	// Handle keyboard input based on state
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Global keys first
		if keyMsg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// State-specific keyboard handling
		switch m.state {
		case StatePortSearch:
			// Handle Enter key for search
			if keyMsg.Type == tea.KeyEnter {
				return m.handlePortSearchKeys(keyMsg)
			}
			// Let textinput handle all other keys
			m.searchInput, cmd = m.searchInput.Update(msg)
			// Clear error when user types
			if m.err != nil {
				m.err = nil
			}
			return m, cmd

		case StateDisplay:
			// Handle Enter key for new search
			if keyMsg.Type == tea.KeyEnter {
				return m.handlePortSearchKeys(keyMsg)
			}
			// Let textinput handle all other keys (for search bar)
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}
	}

	// Update textinput for non-keyboard messages
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}


// handlePortSearchKeys handles keyboard input in port search state
func (m Model) handlePortSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Perform search
		query := m.searchInput.Value()
		if query == "" {
			return m, nil
		}

		ctx := context.Background()
		results, err := m.portClient.SearchByLocation(ctx, query)
		if err != nil {
			// Stay in search state, show error
			m.err = err
			return m, nil
		}

		// Clear any previous errors
		m.err = nil
		m.searchResults = results

		// If no results, show error
		if len(results) == 0 {
			m.err = fmt.Errorf("no stations found for '%s'", query)
			return m, nil
		}

		// Select first result
		m.currentPort = &results[0]
		m.state = StateDisplay
		m.loadingWeather = true
		m.loadingTides = true
		m.loadingAlerts = true
		// Clear old data
		m.weather = nil
		m.forecast = nil
		m.tides = nil
		m.alerts = nil
		// Clear search input for next search
		m.searchInput.SetValue("")
		// Keep search input focused for next search
		m.searchInput.Focus()
		// Fetch all data for this port
		return m, tea.Batch(
			fetchAllData(m.currentPort, m.weatherClient, m.tideClient, m.alertClient),
			textinput.Blink,
		)
	}

	return m, nil
}


// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.state {
	case StatePortSearch:
		return m.viewPortSearch()
	case StateLoading:
		return m.viewLoading()
	case StateDisplay:
		return m.viewDisplay()
	case StateError:
		return m.viewError()
	}

	return ""
}

// viewPortSearch renders the port search view
func (m Model) viewPortSearch() string {
	// Title
	title := titleStyle.Render("⚓ Mariner TUI")
	subtitle := mutedStyle.Render("NOAA Marine Weather & Tide Information")

	// Search box with border
	searchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(54).
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
	examples := mutedStyle.Render("Examples: Chatham, 02633, Woods Hole, Seattle, MA")

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

// viewLoading renders the loading view
func (m Model) viewLoading() string {
	s := "Loading weather and tide data"
	if m.currentPort != nil {
		s += fmt.Sprintf(" for %s, %s", m.currentPort.Name, m.currentPort.State)
	}
	s += "...\n\n"

	if m.loadingWeather {
		s += "⏳ Fetching weather data\n"
	} else {
		s += "✓ Weather data loaded\n"
	}

	if m.loadingTides {
		s += "⏳ Fetching tide data\n"
	} else {
		s += "✓ Tide data loaded\n"
	}

	if m.loadingAlerts {
		s += "⏳ Fetching alerts\n"
	} else {
		s += "✓ Alerts loaded\n"
	}

	return s
}

// viewDisplay renders the main display - simple vertical layout
func (m Model) viewDisplay() string {
	if m.currentPort == nil {
		return "No port selected"
	}

	var sections []string

	// Header
	header := titleStyle.Render(fmt.Sprintf("⚓ %s, %s", m.currentPort.Name, m.currentPort.State))
	sections = append(sections, header, "")

	// Weather section
	sections = append(sections,
		labelStyle.Render("━━━ WEATHER ━━━"),
		m.renderWeatherSimple(),
		"",
	)

	// Tides section
	sections = append(sections,
		labelStyle.Render("━━━ TIDES ━━━"),
		m.renderTideSimple(),
		"",
	)

	// Alerts section
	sections = append(sections,
		labelStyle.Render("━━━ ALERTS ━━━"),
		m.renderAlertSimple(),
		"",
	)

	// Search bar at bottom (always visible)
	searchLabel := mutedStyle.Render("Search: ")
	searchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(50).
		Render(m.searchInput.View())
	searchBar := lipgloss.JoinHorizontal(lipgloss.Left, searchLabel, searchBox)

	// Help text
	help := helpStyle.Render("Enter: Search for new location • Ctrl+C: Quit")

	sections = append(sections, searchBar, "", help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// viewError renders an error view
func (m Model) viewError() string {
	s := "Error occurred:\n\n"
	if m.err != nil {
		s += m.err.Error()
	} else {
		s += "Unknown error"
	}
	s += "\n\nPress Q to quit"
	return s
}

// Setter methods for demo purposes

// SetCurrentPort sets the current port
func (m *Model) SetCurrentPort(port *models.Port) {
	m.currentPort = port
}

// SetWeather sets the weather data
func (m *Model) SetWeather(weather *models.MarineConditions) {
	m.weather = weather
}

// SetForecast sets the forecast data
func (m *Model) SetForecast(forecast *models.ThreeDayForecast) {
	m.forecast = forecast
}

// SetTides sets the tide data
func (m *Model) SetTides(tides *models.TideData) {
	m.tides = tides
}

// SetAlerts sets the alert data
func (m *Model) SetAlerts(alerts *models.AlertData) {
	m.alerts = alerts
}

// SetState sets the application state
func (m *Model) SetState(state AppState) {
	m.state = state
}
