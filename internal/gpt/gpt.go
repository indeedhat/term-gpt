package gpt

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/term"
)

const (
	textAreaHeight   = 3
	chatHistoryWidth = 0.25
)

type focusedElement string

const (
	elemTextArea    focusedElement = "ta"
	elemChatHistory focusedElement = "ch"
)

type Model struct {
	chatHistory     viewport.Model
	chatHistoryList list.Model
	chatVp          viewport.Model
	textarea        textarea.Model
	spinner         spinner.Model

	// Chat concains the message history for this chat session
	chat    *chatLog
	client  *openai.Client
	ctx     context.Context
	cancel  context.CancelFunc
	waiting bool

	windowHeight int
	windowWidth  int

	focus focusedElement

	program *tea.Program
}

// New creates a new model for the bubble tea tui
func New(client *openai.Client) *Model {
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))

	ta := textarea.New()
	ta.Placeholder = "Write your message..."
	// ta.Prompt = "| "
	ta.CharLimit = 1000

	ta.Focus()
	ta.SetWidth(width)
	ta.SetHeight(textAreaHeight)

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	sp.Spinner = spinner.Dot

	historyWidth := int(math.Floor(float64(width) * chatHistoryWidth))

	// i have no idea why i need to do textAreaHeight*2 but it works
	ch := viewport.New(historyWidth, height-textAreaHeight*2)
	ch.Style = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())
	ch.SetContent(lipgloss.NewStyle().Width(width).Render(" "))

	chList := list.New(testItems(), list.NewDefaultDelegate(), historyWidth-2, height-textAreaHeight*2-2)
	chList.Title = "Chat History"

	vp := viewport.New(width-historyWidth, height-textAreaHeight*2)
	vp.Style = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	ctx, cancel := context.WithCancel(context.Background())

	m := &Model{
		textarea:        ta,
		chatHistory:     ch,
		chatHistoryList: chList,
		chatVp:          vp,
		spinner:         sp,
		chat:            newChatLog(),
		client:          client,
		ctx:             ctx,
		cancel:          cancel,
		windowWidth:     width,
		windowHeight:    height,
		focus:           elemTextArea,
	}

	m.updateViewportContent("Welcom to term-gpt!")

	return m
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick, tick)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		taCmd   tea.Cmd
		vpCmd   tea.Cmd
		chCmd   tea.Cmd
		chLiCmd tea.Cmd
	)

	switch m.focus {
	case elemTextArea:
		m.textarea, taCmd = m.textarea.Update(msg)
	case elemChatHistory:
		m.chatHistoryList, chLiCmd = m.chatHistoryList.Update(msg)
	}
	m.chatVp, vpCmd = m.chatVp.Update(msg)
	m.chatHistory, chCmd = m.chatHistory.Update(msg)

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

	case chatResultMsg:
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
		case tea.KeyTab:
			if m.focus == elemTextArea {
				m.textarea.Blur()
				m.focus = elemChatHistory
			} else {
				m.textarea.Focus()
				m.focus = elemTextArea
			}
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

	return m, tea.Batch(taCmd, vpCmd, chCmd, chLiCmd)
}

// View implements tea.Model.
func (m *Model) View() string {
	var textarea string
	if m.waiting {
		textarea = fmt.Sprintf(" %s GPT is thinking...", m.spinner.View())
	} else {
		textarea = m.textarea.View()
	}

	m.chatHistory.SetContent(m.chatHistoryList.View())

	return fmt.Sprintf(
		"\n%s\n\n%s\n\n",
		lipgloss.JoinHorizontal(lipgloss.Top, m.chatVp.View(), m.chatHistory.View()),
		textarea,
	)
}

// handleWindowResize updates the size of the windows containing elements based on the current
// terminal window size
func (m *Model) handleWindowResize(w, h int) {
	m.windowWidth = h
	m.windowWidth = w

	historyWidth := int(math.Floor(float64(w) * chatHistoryWidth))
	m.chatHistory.Width = historyWidth
	m.chatHistory.Height = h - textAreaHeight

	m.chatVp.Height = h - textAreaHeight
	m.chatVp.Width = w - historyWidth

	m.textarea.SetWidth(w)

	m.updateViewportContent(m.chat.Render())
}

func (m *Model) updateViewportContent(text string) {
	m.chatVp.SetContent(lipgloss.NewStyle().Width(m.windowWidth).Render(text))
	m.chatVp.GotoBottom()
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
		m.program.Send(chatResultMsg{err: err})
		return
	}

	m.program.Send(chatResultMsg{message: resp.Choices[0].Message.Content})
}
