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
	"github.com/indeedhat/term-gpt/internal/store"
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

	// Chat concains the message history for this activeChat session
	activeChat chatLog
	client     *openai.Client
	ctx        context.Context
	cancel     context.CancelFunc
	waiting    bool

	windowHeight int
	windowWidth  int

	focus focusedElement

	program *tea.Program
	repo    store.ChatHistoryRepo
}

// New creates a new model for the bubble tea tui
func New(repo store.ChatHistoryRepo, client *openai.Client) *Model {
	// query environment
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))

	// load data
	activeChat := newChatLog()
	chatHistory := append(
		[]store.ChatHistoryMeta{activeChat.history.ChatHistoryMeta},
		repo.List()...,
	)

	// setup ui components
	ta := textarea.New()
	ta.Placeholder = "Write your message..."
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

	chList := list.New(historyList(chatHistory), list.NewDefaultDelegate(), historyWidth-2, height-textAreaHeight*2-2)
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
		client:          client,
		ctx:             ctx,
		cancel:          cancel,
		windowWidth:     width,
		windowHeight:    height,
		focus:           elemTextArea,
		activeChat:      activeChat,
		repo:            repo,
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
	uiCmd := m.updateUiComponents(msg)

	switch msg := msg.(type) {
	case *tea.Program:
		m.program = msg
	case spinMsg:
		m.waiting = true
	case chatResultMsg:
		m.handleChatResultMsg(msg)
	case tea.KeyMsg:
		if cmd := m.handleKeyMsg(msg); cmd != nil {
			return m, cmd
		}
	case tickMsg:
		return m, m.handleTickMsg(msg)
	case error:
		m.cancel()
		return m, tea.Quit
	}

	return m, uiCmd
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

// updateUiComponents handles passing the tea.Msg to all the update methods of the active ui elements
// in order to update their state
func (m *Model) updateUiComponents(msg tea.Msg) tea.Cmd {
	var (
		taCmd   tea.Cmd
		vpCmd   tea.Cmd
		chCmd   tea.Cmd
		chLiCmd tea.Cmd
		spinCmd tea.Cmd
	)

	switch m.focus {
	case elemTextArea:
		m.textarea, taCmd = m.textarea.Update(msg)
	case elemChatHistory:
		m.chatHistoryList, chLiCmd = m.chatHistoryList.Update(msg)
	}
	m.chatVp, vpCmd = m.chatVp.Update(msg)
	m.chatHistory, chCmd = m.chatHistory.Update(msg)
	m.spinner, spinCmd = m.spinner.Update(msg)

	return tea.Batch(taCmd, vpCmd, chCmd, chLiCmd, spinCmd)
}

// handleChatResultMsg takes the chat response from the open ai API and inserts it into the
// chat log
func (m *Model) handleChatResultMsg(msg chatResultMsg) {
	m.waiting = false
	if msg.err != nil {
		msg.message = fmt.Sprintf("Error: %s", msg.err.Error())
	}

	m.activeChat.history.ChatLog = append(m.activeChat.history.ChatLog, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: msg.message,
	})

	m.saveCurrentChat()
	m.updateViewportContent(m.activeChat.Render())
}

// handleTickMsg handles the update tick for the UI
func (m *Model) handleTickMsg(msg tickMsg) tea.Cmd {
	w, h, _ := term.GetSize(int(os.Stdout.Fd()))
	if w != m.windowWidth || h != m.windowHeight {
		m.handleWindowResize(w, h)
		return tea.Batch(tick, windowResize(w, h))
	}

	return tick
}

// handleKeyMsg handles the side effects of any defined tea.KeyMsg key presses
func (m *Model) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyCtrlC:
		fmt.Println("Goodbye :)")
		m.cancel()
		return tea.Quit
	case tea.KeyTab:
		if m.focus == elemTextArea {
			m.textarea.Blur()
			m.focus = elemChatHistory
		} else {
			m.textarea.Focus()
			m.focus = elemTextArea
		}
	case tea.KeyEnter:
		m.activeChat.history.ChatLog = append(m.activeChat.history.ChatLog, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: m.textarea.Value(),
		})

		m.saveCurrentChat()
		m.textarea.Reset()
		m.updateViewportContent(m.activeChat.Render())

		go sendGptRequest(m)
	}

	return nil
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

	m.updateViewportContent(m.activeChat.Render())
}

// updateViewportContent fills the chat viewport with rendered messages constrained to the size
// of the viewport
func (m *Model) updateViewportContent(text string) {
	m.chatVp.SetContent(lipgloss.NewStyle().Width(m.windowWidth).Render(text))
	m.chatVp.GotoBottom()
}

func (m *Model) saveCurrentChat() {
	if m.activeChat.history.Id == 0 {
		_ = m.repo.Create(m.activeChat.history)
	} else {
		_ = m.repo.Update(m.activeChat.history)
	}
	m.chatHistoryList.SetItems(historyList(m.repo.List()))
}

var _ tea.Model = (*Model)(nil)

// sendGptRequest sends off a completion request to the chat gpt api
func sendGptRequest(m *Model) {
	m.program.Send(spinMsg(true))

	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		Messages:  m.activeChat.history.ChatLog,
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
