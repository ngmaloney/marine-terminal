package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ngmaloney/mariner-tui/internal/models"
)

func TestNewModel(t *testing.T) {
	m := NewModel()

	if m.state != StatePortSearch {
		t.Errorf("NewModel() state = %v, want StatePortSearch", m.state)
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
	m.currentPort = &models.Port{Name: "Test", State: "WA"}

	// Tab key should be handled by textinput (no pane switching)
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updatedModel.(Model)

	// Should stay in display state
	if m.state != StateDisplay {
		t.Errorf("After tab, state = %v, want StateDisplay", m.state)
	}

	// Typing should work in display state
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)

	if m.searchInput.Value() != "a" {
		t.Errorf("Expected search input to accept typing in display state, got '%s'", m.searchInput.Value())
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
	if m.state != StatePortSearch {
		t.Errorf("Expected to remain in StatePortSearch, got %v", m.state)
	}

	// Should not have any current port
	if m.currentPort != nil {
		t.Error("Expected currentPort to be nil")
	}
}

// TestPersistentSearchInDisplay verifies search input works in display state
func TestPersistentSearchInDisplay(t *testing.T) {
	m := NewModel()
	m.state = StateDisplay
	m.currentPort = &models.Port{ID: "test", Name: "Test Port", State: "MA"}

	// Verify search input is focused and empty
	if !m.searchInput.Focused() {
		t.Error("Expected search input to be focused in display state")
	}

	// Type in search input while in display state
	for _, char := range "Seattle" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	if m.searchInput.Value() != "Seattle" {
		t.Errorf("Expected search input to be 'Seattle', got '%s'", m.searchInput.Value())
	}

	// Should still be in display state (not transitioned to search state)
	if m.state != StateDisplay {
		t.Errorf("Expected to remain in StateDisplay, got %v", m.state)
	}

	// Pressing Enter should trigger new search and clear input
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(enterMsg)
	m = updatedModel.(Model)

	// Should stay in display state (or transition to it with new port)
	if m.state != StateDisplay {
		t.Errorf("Expected StateDisplay after search, got %v", m.state)
	}

	// Search input should be cleared after search
	if m.searchInput.Value() != "" {
		t.Errorf("Expected search input to be cleared after search, got '%s'", m.searchInput.Value())
	}
}

func TestModel_View_States(t *testing.T) {
	tests := []struct {
		name  string
		state AppState
	}{
		{"port search", StatePortSearch},
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
				m.currentPort = &models.Port{
					Name:  "Test Port",
					State: "CA",
				}
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
	if StatePortSearch != 0 {
		t.Errorf("StatePortSearch = %d, want 0", StatePortSearch)
	}
	if StateLoading != 1 {
		t.Errorf("StateLoading = %d, want 1", StateLoading)
	}
	if StateDisplay != 2 {
		t.Errorf("StateDisplay = %d, want 2", StateDisplay)
	}
	if StateError != 3 {
		t.Errorf("StateError = %d, want 3", StateError)
	}
}

func TestActivePane_Constants(t *testing.T) {
	if PaneWeather != 0 {
		t.Errorf("PaneWeather = %d, want 0", PaneWeather)
	}
	if PaneTides != 1 {
		t.Errorf("PaneTides = %d, want 1", PaneTides)
	}
	if PaneAlerts != 2 {
		t.Errorf("PaneAlerts = %d, want 2", PaneAlerts)
	}
}
