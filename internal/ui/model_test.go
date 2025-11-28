package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ngmaloney/marine-terminal/internal/zonelookup"
)

func TestNewModel(t *testing.T) {
	m := NewModel()

	if m.state != StateSearch {
		t.Errorf("NewModel() state = %v, want StateSearch", m.state)
	}

	if m.activePane != PaneWeather {
		t.Errorf("NewModel() activePane = %v, want PaneWeather", m.activePane)
	}
}

func TestModel_Update_WindowSize(t *testing.T) {
	m := NewModel()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	if m.width != 120 {
		t.Errorf("After WindowSizeMsg, width = %d, want 120", m.width)
	}

	if m.height != 40 {
		t.Errorf("After WindowSizeMsg, height = %d, want 40", m.height)
	}
}

func TestModel_Update_ErrorMsg(t *testing.T) {
	m := NewModel()
	testErr := errMsg{err: tea.ErrProgramKilled}

	updatedModel, _ := m.Update(testErr)
	m = updatedModel.(Model)

	if m.state != StateError {
		t.Errorf("After errMsg, state = %v, want StateError", m.state)
	}

	if m.err == nil {
		t.Error("After errMsg, err should not be nil")
	}
}

func TestModel_CtrlC_Quits(t *testing.T) {
	m := NewModel()

	msg := tea.KeyMsg{Type: tea.KeyCtrlC, Runes: []rune{'c'}}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("Expected Ctrl+C to return quit command")
	}
}

func TestModel_DisplayStateKeyHandling(t *testing.T) {
	m := NewModel()
	m.state = StateDisplay
	m.selectedZone = &zonelookup.ZoneInfo{Code: "ANZ251", Name: "Cape Cod Bay"}

	// Tab key should cycle through panes
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updatedModel.(Model)

	// Should stay in display state
	if m.state != StateDisplay {
		t.Errorf("After tab, state = %v, want StateDisplay", m.state)
	}

	// Active pane should have changed
	if m.activePane != PaneAlerts {
		t.Errorf("After tab, activePane = %v, want PaneAlerts", m.activePane)
	}
}

// TestTextInputHandling verifies that text input works correctly
func TestTextInputHandling(t *testing.T) {
	m := NewModel()

	// Verify search input is focused
	if !m.searchInput.Focused() {
		t.Error("Expected search input to be focused initially")
	}

	// Test typing a character
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	if m.searchInput.Value() != "C" {
		t.Errorf("Expected search input to be 'C', got '%s'", m.searchInput.Value())
	}

	// Test typing multiple characters
	for _, char := range "hatham" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	if m.searchInput.Value() != "Chatham" {
		t.Errorf("Expected search input to be 'Chatham', got '%s'", m.searchInput.Value())
	}

	// Test backspace
	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)

	if m.searchInput.Value() != "Chatha" {
		t.Errorf("Expected search input to be 'Chatha' after backspace, got '%s'", m.searchInput.Value())
	}

	// Test space (using rune, not KeySpace type)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)

	if m.searchInput.Value() != "Chatha " {
		t.Errorf("Expected search input to include space, got '%s'", m.searchInput.Value())
	}
}

// TestErrorClearingOnInput verifies that errors are cleared when user types
func TestErrorClearingOnInput(t *testing.T) {
	m := NewModel()

	// Set an error
	m.err = tea.ErrProgramKilled

	// Verify error is set
	if m.err == nil {
		t.Fatal("Test setup failed - error should be set")
	}

	// Simulate typing to clear error
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	// Error should be cleared
	if m.err != nil {
		t.Error("Expected error to be cleared when user types")
	}
}

// TestEnterKeyWithEmptyInput verifies that pressing Enter with empty input does nothing
func TestEnterKeyWithEmptyInput(t *testing.T) {
	m := NewModel()

	// Press Enter with empty input
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	// Should still be in search state
	if m.state != StateSearch {
		t.Errorf("Expected to remain in StateSearch, got %v", m.state)
	}

	// Should not have any selected zone
	if m.selectedZone != nil {
		t.Error("Expected selectedZone to be nil")
	}
}

// TestSearchAgainFromDisplay verifies 'S' key returns to search
func TestSearchAgainFromDisplay(t *testing.T) {
	m := NewModel()
	m.state = StateDisplay
	m.selectedZone = &zonelookup.ZoneInfo{Code: "ANZ251", Name: "Cape Cod Bay"}

	// Press 'S' to search again
	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	updatedModel, _ := m.Update(sMsg)
	m = updatedModel.(Model)

	// Should transition to search state
	if m.state != StateSearch {
		t.Errorf("Expected StateSearch after 's' key, got %v", m.state)
	}

	// Data should be cleared
	if m.selectedZone != nil {
		t.Error("Expected selectedZone to be cleared")
	}
}

func TestModel_View_States(t *testing.T) {
	tests := []struct {
		name  string
		state AppState
	}{
		{"search", StateSearch},
		{"zone list", StateZoneList},
		{"loading", StateLoading},
		{"display", StateDisplay},
		{"error", StateError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			m.state = tt.state
			m.width = 80
			m.height = 24

			if tt.state == StateDisplay {
				m.selectedZone = &zonelookup.ZoneInfo{
					Code: "ANZ251",
					Name: "Cape Cod Bay",
				}
			}

			if tt.state == StateZoneList {
				m.zones = []zonelookup.ZoneInfo{
					{Code: "ANZ251", Name: "Cape Cod Bay", Distance: 5.2},
				}
				m.zoneList = createZoneList(m.zones, 80, 20)
			}

			view := m.View()
			if view == "" {
				t.Errorf("View() returned empty string for state %v", tt.state)
			}
		})
	}
}

func TestModel_View_InitialLoading(t *testing.T) {
	m := NewModel()
	view := m.View()

	if view != "Loading..." {
		t.Errorf("View() before window size = %q, want 'Loading...'", view)
	}
}

func TestAppState_Constants(t *testing.T) {
	if StateSearch != 0 {
		t.Errorf("StateSearch = %d, want 0", StateSearch)
	}
	if StateZoneList != 1 {
		t.Errorf("StateZoneList = %d, want 1", StateZoneList)
	}
	if StateLoading != 2 {
		t.Errorf("StateLoading = %d, want 2", StateLoading)
	}
	if StateDisplay != 3 {
		t.Errorf("StateDisplay = %d, want 3", StateDisplay)
	}
	if StateError != 4 {
		t.Errorf("StateError = %d, want 4", StateError)
	}
}

func TestActivePane_Constants(t *testing.T) {
	if PaneWeather != 0 {
		t.Errorf("PaneWeather = %d, want 0", PaneWeather)
	}
	if PaneAlerts != 1 {
		t.Errorf("PaneAlerts = %d, want 1", PaneAlerts)
	}
}
