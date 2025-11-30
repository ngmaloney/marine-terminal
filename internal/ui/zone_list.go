package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ngmaloney/marine-terminal/internal/zonelookup"
)

// zoneItem wraps a ZoneInfo for use in a list
type zoneItem struct {
	zone zonelookup.ZoneInfo
}

// FilterValue implements list.Item
func (z zoneItem) FilterValue() string {
	return z.zone.Code + " " + z.zone.Name
}

// zoneDelegate implements list.ItemDelegate for rendering zone items
type zoneDelegate struct{}

func (d zoneDelegate) Height() int                               { return 1 }
func (d zoneDelegate) Spacing() int                              { return 0 }
func (d zoneDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd   { return nil }
func (d zoneDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	z, ok := item.(zoneItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s - %s", z.zone.Code, z.zone.Name)
	if z.zone.Distance > 0 {
		str += fmt.Sprintf(" (%.1f mi)", z.zone.Distance)
	}

	var style lipgloss.Style
	if index == m.Index() {
		// Selected state
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")). // Pinkish/Magenta for selection
			PaddingLeft(2).
			Bold(true)
		str = "> " + str
	} else {
		// Normal state
		style = lipgloss.NewStyle().
			PaddingLeft(4)
	}

	fmt.Fprint(w, style.Render(str))
}

// createZoneList creates a list.Model from zone info
func createZoneList(zones []zonelookup.ZoneInfo, width, height int) list.Model {
	items := make([]list.Item, len(zones))
	for i, zone := range zones {
		items[i] = zoneItem{zone: zone}
	}

	// Use custom delegate to ensure 1-line height and avoid rendering artifacts
	l := list.New(items, zoneDelegate{}, width, height)
	l.Title = "Select a Marine Zone"
	l.SetShowHelp(true)
	l.SetFilteringEnabled(false)

	return l
}
