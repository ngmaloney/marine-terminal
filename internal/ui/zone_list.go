package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/ngmaloney/mariner-tui/internal/zonelookup"
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
	return fmt.Sprintf("%s - %s", z.zone.Code, z.zone.Name)
}

// Description implements list.DefaultItem
func (z zoneItem) Description() string {
	return fmt.Sprintf("%.1f miles away", z.zone.Distance)
}

// createZoneList creates a list.Model from zone info
func createZoneList(zones []zonelookup.ZoneInfo, width, height int) list.Model {
	items := make([]list.Item, len(zones))
	for i, zone := range zones {
		items[i] = zoneItem{zone: zone}
	}

	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = "Select a Marine Zone"
	l.SetShowHelp(true)
	l.SetFilteringEnabled(false)

	return l
}
