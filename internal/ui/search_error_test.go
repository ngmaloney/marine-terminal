package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestSearch_ErrorRecovery tests that users can recover from search errors
func TestSearch_ErrorRecovery(t *testing.T) {
	m := NewModel()

	// Step 1: User types invalid search
	for _, char := range "InvalidCity123" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	if m.searchInput.Value() != "InvalidCity123" {
		t.Errorf("searchInput.Value() = %s, want 'InvalidCity123'", m.searchInput.Value())
	}

	// Step 2: User presses Enter - search fails
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(enterMsg)
	m = updatedModel.(Model)

	// Should have error
	if m.err == nil {
		t.Error("Expected error for invalid search")
	}

	// Should still be in search state (not error state)
	if m.state != StatePortSearch {
		t.Errorf("state = %v, want StatePortSearch (should stay in search on error)", m.state)
	}

	// Step 3: User starts typing again - error should clear
	backspaceMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ = m.Update(backspaceMsg)
	m = updatedModel.(Model)

	if m.err != nil {
		t.Error("Error should be cleared when user modifies search")
	}

	// Step 4: Clear input and type valid search
	m.searchInput.SetValue("") // Clear previous query
	for _, char := range "Seattle" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	// Step 5: User presses Enter - should succeed
	enterMsg = tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(enterMsg)
	m = updatedModel.(Model)

	// Should have no error
	if m.err != nil {
		t.Errorf("Unexpected error after valid search: %v", m.err)
	}

	// Should transition to display
	if m.state != StateDisplay {
		t.Errorf("state = %v, want StateDisplay", m.state)
	}

	// Should have selected port
	if m.currentPort == nil {
		t.Error("Expected port to be selected")
	}

	// Should return command to fetch data
	if cmd == nil {
		t.Error("Expected command to fetch data")
	}
}

// TestSearch_EmptyQueryHandling tests empty search handling
func TestSearch_EmptyQueryHandling(t *testing.T) {
	m := NewModel()

	// Press Enter with empty query
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(enterMsg)
	m = updatedModel.(Model)

	// Should stay in search state
	if m.state != StatePortSearch {
		t.Errorf("state = %v, want StatePortSearch", m.state)
	}

	// Should not have error
	if m.err != nil {
		t.Error("Should not error on empty query, just do nothing")
	}
}

// TestSearch_StateAbbreviation tests state abbreviation search functionality
func TestSearch_StateAbbreviation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API integration test in short mode")
	}

	m := NewModel()

	// Search by state abbreviation
	for _, char := range "MA" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(enterMsg)
	m = updatedModel.(Model)

	// Should find Massachusetts stations
	if m.currentPort == nil {
		t.Fatal("Expected to find port by state abbreviation")
	}

	if m.currentPort.State != "MA" {
		t.Errorf("State MA should find Massachusetts ports, got %s", m.currentPort.State)
	}

	// Should be in display state
	if m.state != StateDisplay {
		t.Errorf("state = %v, want StateDisplay", m.state)
	}
}
