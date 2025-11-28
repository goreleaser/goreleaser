package logext

import (
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/caarlos0/log"
)

var (
	// URL is used to style URLs.
	URL = lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).
		Render

	// Warning is used to style warnings for the user.
	Warning = lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")).
		Bold(true).
		Render

	// Keyword should be used to highlight code.
	Keyword = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ED567A")).
		Background(lipgloss.Color("#242424")).
		Render

	// Faint is fainted, italic, text.
	Faint = lipgloss.NewStyle().
		Italic(true).
		Faint(true).
		Render
)

// Duration logs the given duration if it exceeds the given threshold.
func Duration(start time.Time, threshold time.Duration) {
	if took := time.Since(start).Round(time.Second); took > threshold {
		log.IncreasePadding()
		log.Info(Faint(fmt.Sprintf("took: %s", took)))
		log.DecreasePadding()
	}
}
