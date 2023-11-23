package gpt

import (
	"bytes"

	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"
)

type chatLog struct {
	messages  []openai.ChatCompletionMessage
	nameStyle lipgloss.Style
}

func newChatLog() *chatLog {
	return &chatLog{
		nameStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
	}
}

// Render the chat log to a string
func (c chatLog) Render() string {
	var buf bytes.Buffer

	for _, msg := range c.messages {
		name := "You: "
		if msg.Role == openai.ChatMessageRoleSystem {
			name = "GPT: "
		}

		buf.WriteString(c.nameStyle.Render(name) + msg.Content + "\n\n")
	}

	return buf.String()
}
