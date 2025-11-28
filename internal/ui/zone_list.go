package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
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

// Title implements list.DefaultItem
func (z zoneItem) Title() string {
	return fmt.Sprintf("%s - %s (%.1f mi)", z.zone.Code, z.zone.Name, z.zone.Distance)
}

// Description implements list.DefaultItem
func (z zoneItem) Description() string {
	return "" // Empty to avoid duplicate display
}

// createZoneList creates a list.Model from zone info
func createZoneList(zones []zonelookup.ZoneInfo, width, height int) list.Model {
	items := make([]list.Item, len(zones))
	for i, zone := range zones {
		items[i] = zoneItem{zone: zone}
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1) // Only title line needed now
	delegate.SetSpacing(0)
	delegate.ShowDescription = false // Don't show description since it's empty

	l := list.New(items, delegate, width, height)
	l.Title = "Select a Marine Zone"
	l.SetShowHelp(true)
	l.SetFilteringEnabled(false)

	return l
}
