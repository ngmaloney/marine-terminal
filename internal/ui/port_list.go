package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/ngmaloney/marine-terminal/internal/models"
)

// portItem wraps a Port for use in a list
type portItem struct {
	port models.Port
}

// FilterValue implements list.Item
func (p portItem) FilterValue() string {
	return p.port.Name
}

// Title implements list.DefaultItem
func (p portItem) Title() string {
	return p.port.Name
}

// Description implements list.DefaultItem
func (p portItem) Description() string {
	desc := p.port.StationID
	if p.port.City != "" || p.port.State != "" {
		desc += fmt.Sprintf(" • %s, %s", p.port.City, p.port.State)
	}
	if p.port.Zipcode != "" {
		desc += fmt.Sprintf(" %s", p.port.Zipcode)
	}
	return desc
}

// createPortList creates a list.Model from ports
func createPortList(ports []models.Port, width, height int) list.Model {
	items := make([]list.Item, len(ports))
	for i, port := range ports {
		items[i] = portItem{port: port}
	}

	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = "Select a Saved Port"
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)

	return l
}
