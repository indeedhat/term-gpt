package gpt

import "github.com/indeedhat/term-gpt/internal/store"

// updateChatList replaces the chat history with a fresh up to date version from the database
func updateChatList(m *Model) {
	m.chatHistoryList.SetItems(historyList(m.repo.List()))
	m.chatHistoryList.Select(0)
}

// loadChat loads the full chat by id into the models activeChat struct
func loadChat(m *Model) {
	item := m.chatHistoryList.SelectedItem().(store.ChatHistoryMeta)
	if item.Id == 0 {
		m.activeChat = newChatLog()
	} else {
		m.activeChat.history = m.repo.Find(item.Id)
	}
}

// saveChat saves the active chat to the database
// this will also update the ui with the corrected history list as specified in the database
func saveChat(m *Model) error {
	if m.activeChat.history.Id == 0 {
		if err := m.repo.Create(m.activeChat.history); err != nil {
			return err
		}
	} else {
		if err := m.repo.Update(m.activeChat.history); err != nil {
			return err
		}
	}

	return nil
}
