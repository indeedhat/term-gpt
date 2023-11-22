package gpt

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"
)

type chatResult struct {
	err     error
	message string
}

type spinMsg bool

type Model struct {
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	// Chat concains the message history for this chat session
	chat    *chatLog
	client  *openai.Client
	ctx     context.Context
	cancel  context.CancelFunc
	waiting bool

	program *tea.Program
}

// New creates a new model for the bubble tea tui
func New(client *openai.Client) Model {
	ta := textarea.New()
	ta.Placeholder = "Write your message..."
	ta.Prompt = "| "
	ta.CharLimit = 1000

	ta.Focus()
	ta.SetWidth(50)
	ta.SetHeight(3)

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	sp.Spinner = spinner.Dot

	vp := viewport.New(50, 10)
	vp.SetContent("Welcom to term-gpt!")

	ctx, cancel := context.WithCancel(context.Background())

	return Model{
		textarea: ta,
		viewport: vp,
		spinner:  sp,
		chat:     newChatLog(),
		client:   client,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		taCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, taCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case *tea.Program:
		m.program = msg

	case spinMsg:
		m.waiting = true

	case chatResult:
		m.waiting = false
		if msg.err != nil {
			msg.message = fmt.Sprintf("Error: %s", msg.err.Error())
		}
		m.chat.messages = append(m.chat.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: msg.message,
		})
		m.viewport.SetContent(m.chat.Render())
		m.viewport.GotoBottom()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			fmt.Println("Goodbye :)")
			m.cancel()
			return m, tea.Quit
		case tea.KeyEnter:
			m.chat.messages = append(m.chat.messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: m.textarea.Value(),
			})

			m.viewport.SetContent(m.chat.Render())
			m.textarea.Reset()
			m.viewport.GotoBottom()

			go sendGptRequest(m)
		}

	case error:
		fmt.Println(msg.Error())
		m.cancel()
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, tea.Batch(taCmd, vpCmd)
}

// View implements tea.Model.
func (m Model) View() string {
	var spinner string
	if m.waiting {
		spinner = m.spinner.View() + "\n\n"
	}

	return fmt.Sprintf(
		"%s\n\n%s%s\n\n",
		m.viewport.View(),
		spinner,
		m.textarea.View(),
	)
}

var _ tea.Model = (*Model)(nil)

// sendGptRequest sends off a completion request to the chat gpt api
func sendGptRequest(m Model) {
	m.program.Send(spinMsg(true))

	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		Messages:  m.chat.messages,
		MaxTokens: 2000,
	}
	resp, err := m.client.CreateChatCompletion(m.ctx, req)

	if err != nil {
		fmt.Print(err)
		m.program.Send(chatResult{err: err})
		return
	}

	m.program.Send(chatResult{message: resp.Choices[0].Message.Content})
}
