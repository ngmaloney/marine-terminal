package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette
	colorPrimary   = lipgloss.Color("#00BFFF") // Deep sky blue
	colorSecondary = lipgloss.Color("#87CEEB") // Sky blue
	colorDanger    = lipgloss.Color("#FF6B6B") // Red for alerts
	colorWarning   = lipgloss.Color("#FFD93D") // Yellow for warnings
	colorSuccess   = lipgloss.Color("#6BCF7F") // Green
	colorMuted     = lipgloss.Color("#6C757D") // Gray
	colorBorder    = lipgloss.Color("#4A90E2") // Border blue

	// Base styles
	baseStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Title styles (no padding - paneStyle already has padding)
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	activeTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colorPrimary)

	// Pane styles
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2).
			MarginRight(1)

	activePaneStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2).
			MarginRight(1)

	// Content styles
	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(true)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	// Alert severity styles
	alertExtremeStyle = lipgloss.NewStyle().
				Foreground(colorDanger).
				Bold(true)

	alertSevereStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF8C42")).
				Bold(true)

	alertModerateStyle = lipgloss.NewStyle().
				Foreground(colorWarning).
				Bold(true)

	alertMinorStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	// Help text style
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(1, 0)

	// Utility styles
	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	// Section header styles
	sectionHeaderStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				Padding(0, 1).
				MarginTop(1)

	boxHeaderStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 0, 1, 0) // Padding bottom 1

	sectionBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2).
			MarginBottom(1)
)
