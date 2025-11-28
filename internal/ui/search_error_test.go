package ui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ngmaloney/marine-terminal/internal/geocoding"
)

// TestSearch_ErrorRecovery tests that users can recover from geocoding errors
func TestSearch_ErrorRecovery(t *testing.T) {
	m := NewModel()

	// Step 1: User types invalid location
	for _, char := range "InvalidCity123" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	if m.searchInput.Value() != "InvalidCity123" {
		t.Errorf("searchInput.Value() = %s, want 'InvalidCity123'", m.searchInput.Value())
	}

	// Step 2: User presses Enter - triggers geocoding
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(enterMsg)
	m = updatedModel.(Model)

	// Should transition to loading state
	if m.state != StateLoading {
		t.Errorf("state = %v, want StateLoading", m.state)
	}

	// Step 3: Simulate geocoding failure
	geocodeMsg := geocodeMsg{err: fmt.Errorf("location not found")}
	updatedModel, _ = m.Update(geocodeMsg)
	m = updatedModel.(Model)

	// Should have error
	if m.err == nil {
		t.Error("Expected error for failed geocoding")
	}

	// Should transition to error state
	if m.state != StateError {
		t.Errorf("state = %v, want StateError", m.state)
	}

	// Step 4: User starts typing again - should return to search and clear error
	for _, char := range "02633" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	// Error should be cleared
	if m.err != nil {
		t.Error("Error should be cleared when user modifies search")
	}

	// Should return to search state
	if m.state != StateSearch {
		t.Errorf("state = %v, want StateSearch (typing should return to search)", m.state)
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
	if m.state != StateSearch {
		t.Errorf("state = %v, want StateSearch", m.state)
	}

	// Should not have error
	if m.err != nil {
		t.Error("Should not error on empty query, just do nothing")
	}

	// Should not have triggered any command
	if m.location != nil {
		t.Error("Should not have geocoded empty query")
	}
}

// TestSearch_ZipcodeSearch tests zipcode search functionality
func TestSearch_ZipcodeSearch(t *testing.T) {
	m := NewModel()

	// Search by zipcode
	for _, char := range "02633" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(Model)
	}

	if m.searchInput.Value() != "02633" {
		t.Errorf("searchInput.Value() = %s, want '02633'", m.searchInput.Value())
	}

	// Press Enter to trigger geocoding
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(enterMsg)
	m = updatedModel.(Model)

	// Should transition to loading state
	if m.state != StateLoading {
		t.Errorf("state = %v, want StateLoading", m.state)
	}
}

// TestSearch_NoZonesFound tests handling when no zones are found near location
func TestSearch_NoZonesFound(t *testing.T) {
	m := NewModel()
	m.state = StateLoading

	// Simulate successful geocoding
	mockLocation := &geocoding.Location{
		Latitude:  90.0, // North Pole - no marine zones
		Longitude: 0.0,
		Name:      "North Pole",
	}
	geocodeMsg := geocodeMsg{location: mockLocation}
	updatedModel, _ := m.Update(geocodeMsg)
	m = updatedModel.(Model)

	// Location should be set
	if m.location == nil {
		t.Fatal("Expected location to be set")
	}

	// Should trigger zone search (we can't test the full flow without the database,
	// but we can verify the location was stored)
	if m.location.Name != "North Pole" {
		t.Errorf("location.Name = %s, want 'North Pole'", m.location.Name)
	}
}
