package gpt

import (
	"bytes"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/indeedhat/term-gpt/internal/store"
	"github.com/sashabaranov/go-openai"
)

type chatLog struct {
	history   *store.ChatHistory
	nameStyle lipgloss.Style
}

// newChatLog helper for setting up the chat log instance
func newChatLog() chatLog {
	return chatLog{
		nameStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		history: &store.ChatHistory{
			ChatHistoryMeta: store.ChatHistoryMeta{
				ChatTitle: "New Chat",
				UpdatedAt: time.Now(),
			},
		},
	}
}

// Render the chat log to a string
func (c chatLog) Render() string {
	var buf bytes.Buffer

	for _, msg := range c.history.ChatLog {
		name := "You: "
		if msg.Role == openai.ChatMessageRoleSystem {
			name = "GPT: "
		}

		buf.WriteString(c.nameStyle.Render(name) + msg.Content + "\n\n")
	}

	return buf.String()
}

// historyList converts a []store.ChatHistoryMeta slice into a []list.Item slice
// because go interfaces don't play nicely with slices
func historyList(history []store.ChatHistoryMeta) []list.Item {
	l := make([]list.Item, 0, len(history))

	for _, entry := range history {
		l = append(l, entry)
	}

	return l
}
