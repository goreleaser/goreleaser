package logext

import "github.com/charmbracelet/lipgloss"

// Keyword should be used to highlight code.
var Keyword = lipgloss.NewStyle().
	Padding(0, 1).
	Foreground(lipgloss.AdaptiveColor{Light: "#FF4672", Dark: "#ED567A"}).
	Background(lipgloss.AdaptiveColor{Light: "#DDDADA", Dark: "#242424"}).
	Render

var (
	// URL is used to style URLs.
	URL = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render

	// Warning is used to style warnings for the user.
	Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true).Render
)
