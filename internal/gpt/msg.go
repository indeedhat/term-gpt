package gpt

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg struct{}

// tick keeps the update cycle ticking at rate of 5hz
func tick() tea.Msg {
	time.Sleep(time.Second / 5)
	return tickMsg{}
}

// windowResize wraps the windowResizeMsg Cmd forconvenience
func windowResize(w, h int) tea.Cmd {
	return func() tea.Msg {
		return tea.WindowSizeMsg{
			Width:  w,
			Height: h,
		}
	}
}
