package gpt

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/term"
)

const textAreaHeight = 3

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

	windowHeight int
	windowWidth  int

	program *tea.Program
}

// New creates a new model for the bubble tea tui
func New(client *openai.Client) *Model {
	w, h, _ := term.GetSize(int(os.Stdout.Fd()))

	ta := textarea.New()
	ta.Placeholder = "Write your message..."
	// ta.Prompt = "| "
	ta.CharLimit = 1000

	ta.Focus()
	ta.SetWidth(w)
	ta.SetHeight(textAreaHeight)

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	sp.Spinner = spinner.Dot

	vp := viewport.New(w, h-textAreaHeight*2)
	vp.Width = w
	vp.Height = h - textAreaHeight*2
	vp.SetContent(lipgloss.NewStyle().Width(w).Render(fmt.Sprintf("Welcom to term-gpt!\n%dx%d", w, h)))
	vp.Style = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	ctx, cancel := context.WithCancel(context.Background())

	return &Model{
		textarea:     ta,
		viewport:     vp,
		spinner:      sp,
		chat:         newChatLog(),
		client:       client,
		ctx:          ctx,
		cancel:       cancel,
		windowWidth:  w,
		windowHeight: h,
	}
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick, tick)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case tickMsg:
		w, h, _ := term.GetSize(int(os.Stdout.Fd()))
		if w != m.windowWidth || h != m.windowHeight {
			m.handleWindowResize(w, h)
			return m, tea.Batch(tick, windowResize(w, h))
		}

		return m, tick

	case chatResult:
		m.waiting = false
		if msg.err != nil {
			msg.message = fmt.Sprintf("Error: %s", msg.err.Error())
		}
		m.chat.messages = append(m.chat.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: msg.message,
		})
		m.updateViewportContent(m.chat.Render())

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

			m.textarea.Reset()
			m.updateViewportContent(m.chat.Render())

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
func (m *Model) View() string {
	var textarea string
	if m.waiting {
		textarea = fmt.Sprintf(" %s GPT is thinking...", m.spinner.View())
	} else {
		textarea = m.textarea.View()
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n",
		m.viewport.View(),
		textarea,
	)
}

// handleWindowResize updates the size of the windows containing elements based on the current
// terminal window size
func (m *Model) handleWindowResize(w, h int) {
	m.windowWidth = h
	m.windowWidth = w

	m.viewport.Height = h - textAreaHeight
	m.viewport.Width = w

	m.textarea.SetWidth(w)

	m.updateViewportContent(m.chat.Render())
}

func (m *Model) updateViewportContent(text string) {
	m.viewport.SetContent(lipgloss.NewStyle().Width(m.windowWidth).Render(text))
	m.viewport.GotoBottom()
}

var _ tea.Model = (*Model)(nil)

// sendGptRequest sends off a completion request to the chat gpt api
func sendGptRequest(m *Model) {
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
