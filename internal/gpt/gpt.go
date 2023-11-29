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

// focusedElement represents the ui element that the user is currently interacting with and
// defines the keyboard behaviour for that element
//
// for example the textarea will only accept text input when it is focused
type focusedElement string

const (
	elemTextArea    focusedElement = "ta"
	elemChatHistory focusedElement = "ch"
)

type Model struct {
	// chatHistory is the ui element that displays the previous chats
	chatHistory viewport.Model
	// chatHistoryList contains the data model for the above chatHistory element
	chatHistoryList list.Model
	// chatVp represents the ui element that displays the currently active chat history
	chatVp viewport.Model
	// textarea is the ui element that the user types into
	textarea textarea.Model
	// spinner to show when we are waiting for a response from ChatGPT
	spinner spinner.Model

	// Chat concains the message history for this activeChat session
	activeChat chatLog

	// client holds the openai client for
	client *openai.Client

	// ctx is the shared context sent along with web requests an can be used to gracefully close
	// connections early if the app closes while a web request is running
	ctx context.Context
	// cancel stores the cancel function that can be used to close the ctx
	cancel context.CancelFunc

	// waiting tracks if we are currently waiting for a response from the openai API or not
	waiting bool

	// windowHeight stores the height of the terminal from the previous frame
	windowHeight int
	// windowWidth stores the width of the terminal from the previous frame
	windowWidth int

	// focus tracks the element the user is currently focusing
	focus focusedElement

	// program stores the bubble tea program reference
	program *tea.Program
	// repo is the storage repository that persists chat data between sessions
	repo store.ChatHistoryRepo
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
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	uiCmd := m.updateUiComponents(msg)

	switch msg := msg.(type) {
	case *tea.Program:
		m.program = msg
	case tea.WindowSizeMsg:
		m.handleWindowResize()
	case spinMsg:
		m.waiting = true
	case chatResultMsg:
		m.handleChatResultMsg(msg)
	case tea.KeyMsg:
		if cmd := m.handleKeyMsg(msg); cmd != nil {
			return m, cmd
		}
	case error:
		m.cancel()
		return m, tea.Quit
	}

	return m, uiCmd
}

// View implements tea.Model.
// It is called to generate the current frame for the application
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
		curIdx := m.chatHistoryList.Index()
		m.chatHistoryList, chLiCmd = m.chatHistoryList.Update(msg)

		// i suspect i might end up with a race condition problem here when sending gpt
		// requests, if it happens i may need to add a mutex
		if m.chatHistoryList.Index() != curIdx {
			loadChat(m)
			m.updateViewportContent(m.activeChat.Render())
		}
	}

	// m.chatVp, vpCmd = m.chatVp.Update(msg)
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

	saveChat(m)
	updateChatList(m)

	m.updateViewportContent(m.activeChat.Render())
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

		saveChat(m)
		updateChatList(m)

		m.textarea.Reset()
		m.updateViewportContent(m.activeChat.Render())

		go sendGptRequest(m)
	}

	return nil
}

// handleWindowResize updates the size of the windows containing elements based on the current
// terminal window size
func (m *Model) handleWindowResize() {
	w, h, _ := term.GetSize(int(os.Stdout.Fd()))
	if w == m.windowWidth && h == m.windowHeight {
		return
	}

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

var _ tea.Model = (*Model)(nil)
