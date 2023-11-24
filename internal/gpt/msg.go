package gpt

import (
	tea "github.com/charmbracelet/bubbletea"
)

type chatResultMsg struct {
	err     error
	message string
}

type spinMsg bool

// windowResize wraps the windowResizeMsg Cmd forconvenience
func windowResize(w, h int) tea.Cmd {
	return func() tea.Msg {
		return tea.WindowSizeMsg{
			Width:  w,
			Height: h,
		}
	}
}
