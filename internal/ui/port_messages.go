package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ngmaloney/marine-terminal/internal/models"
	"github.com/ngmaloney/marine-terminal/internal/ports"
)

type portsFetchedMsg struct {
	ports []models.Port
	err   error
}

type portSavedMsg struct {
	port *models.Port
	err  error
}

type portFetchedMsg struct {
	port *models.Port
	err  error
}

func fetchSavedPorts(s *ports.Service) tea.Cmd {
	return func() tea.Msg {
		ports, err := s.ListPorts()
		return portsFetchedMsg{ports: ports, err: err}
	}
}

func savePort(s *ports.Service, name, inputLocation, marineZoneCode string) tea.Cmd {
	return func() tea.Msg {
		port, err := s.CreatePort(context.Background(), name, inputLocation, marineZoneCode)
		return portSavedMsg{port: port, err: err}
	}
}

func fetchPortByName(s *ports.Service, name string) tea.Cmd {
	return func() tea.Msg {
		// We can't easily get by name efficiently without adding a method to Service/Repo
		// For now, list all and filter? Or add GetPortByName to Service.
		// Let's rely on ListPorts for now as it's simpler and the list won't be huge.
		ports, err := s.ListPorts()
		if err != nil {
			return portFetchedMsg{err: err}
		}
		for _, p := range ports {
			if p.Name == name {
				return portFetchedMsg{port: &p}
			}
		}
		return portFetchedMsg{err: fmt.Errorf("port not found: %s", name)}
	}
}

type portDeletedMsg struct {
	name string
	err  error
}

func deletePort(s *ports.Service, name string) tea.Cmd {
	return func() tea.Msg {
		err := s.DeletePort(name)
		return portDeletedMsg{name: name, err: err}
	}
}