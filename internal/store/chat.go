package store

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/sashabaranov/go-openai"
)

type ChatLog []openai.ChatCompletionMessage

// Value implements driver.Valuer.
func (l ChatLog) Value() (driver.Value, error) {
	if l == nil {
		return nil, nil
	}

	data, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}

	return string(data), nil
}

// Scan implements sql.Scanner.
func (l *ChatLog) Scan(src any) error {
	switch val := src.(type) {
	case []byte:
		return json.Unmarshal(val, l)
	case string:
		return json.Unmarshal([]byte(val), l)
	default:
		return errors.New("invalid type")
	}
}

var _ driver.Valuer = (*ChatLog)(nil)
var _ sql.Scanner = (*ChatLog)(nil)

type ChatHistory struct {
	ChatHistoryMeta

	ChatLog ChatLog
}

type ChatHistoryMeta struct {
	Id        int
	ChatTitle string
	UpdatedAt time.Time
}

// FilterValue implements list.Item.
func (m ChatHistoryMeta) FilterValue() string {
	return m.ChatTitle
}

// Title returns the private title member
func (m ChatHistoryMeta) Title() string {
	return m.ChatTitle
}

// Description returns the private date member as the list item description
func (m ChatHistoryMeta) Description() string {
	return m.UpdatedAt.Format(time.DateTime)
}

// Ensure that ChatHistoryMeta can be used as a list item by bubbletea
var _ list.Item = (*ChatHistoryMeta)(nil)

type ChatHistoryRepo interface {
	// MigrateSchema creates/updates the database schema for the chat_history table
	MigrateSchema() error
	// Create adds a new entry to the chat_history table
	Create(entry *ChatHistory) error
	// Update updates an existing entry in the chat_history table
	Update(entry *ChatHistory) error
	// List returns a list of all the saved chat logs in the chat_history table
	// It will only return the meta data for each entry, not the chat logs themselves
	List() []ChatHistoryMeta
	// Fild returns a full entry from the chat_history table with the logs included
	Find(id int) *ChatHistory
}

type ChatHistorySqliteRepo struct {
	db *sql.DB
}

// NewChatHistorySqliteRepo sets up the sqlite repository for managing the
// chat history data store
func NewChatHistorySqliteRepo(db *sql.DB) ChatHistorySqliteRepo {
	return ChatHistorySqliteRepo{db}
}

// MigrateSchema implements ChatHistoryRepo.
func (r ChatHistorySqliteRepo) MigrateSchema() error {
	_, err := r.db.Exec(`
        CREATE TABLE IF NOT EXISTS chat_history (
            id INTEGER PRIMARY key AUTOINCREMENT,
            title varchar(100),
            updated_at TEXT,
            chat_log TEXT
        )
    `)

	return err
}

// Create implements ChatHistoryRepo.
func (r ChatHistorySqliteRepo) Create(entry *ChatHistory) error {
	if len(entry.ChatLog) == 0 {
		return errors.New("cannot save an empty chat log")
	}

	title := substr(entry.ChatLog[0].Content, 0, 100)
	entry.ChatTitle = title

	res, err := r.db.Exec(`
        INSERT INTO chat_history (
            title,
            updated_at,
            chat_log
        ) VALUES (
            ?, strftime('%s', 'now'), ?
        )
    `, title, entry.ChatLog)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	tmp := r.Find(int(id))
	if tmp != nil {
		*entry = *tmp
	}

	return nil
}

// Find implements ChatHistoryRepo.
func (r ChatHistorySqliteRepo) Find(id int) *ChatHistory {
	var entry ChatHistory

	row := r.db.QueryRow(`
        SELECT id, title, updated_at, chat_log
        FROM chat_history
        WHERE id = ?
    `, id)
	if row == nil {
		return nil
	}
	var ud int64

	err := row.Scan(&entry.Id, &entry.ChatTitle, &ud, &entry.ChatLog)
	if err != nil {
		return nil
	}
	entry.UpdatedAt = time.Unix(ud, 0)

	return &entry
}

// List implements ChatHistoryRepo.
func (r ChatHistorySqliteRepo) List() []ChatHistoryMeta {
	var entries []ChatHistoryMeta

	rows, err := r.db.Query(`
        SELECT id, title, updated_at
        FROM chat_history
        ORDER BY updated_at DESC
    `)
	if err != nil {
		return nil
	}

	for rows.Next() {
		var (
			entry ChatHistoryMeta
			ud    int64
		)
		if err := rows.Scan(&entry.Id, &entry.ChatTitle, &ud); err != nil {
			continue
		}

		entry.UpdatedAt = time.Unix(ud, 0)
		entries = append(entries, entry)
	}

	return entries
}

// Update implements ChatHistoryRepo.
func (r ChatHistorySqliteRepo) Update(entry *ChatHistory) error {
	_, err := r.db.Exec(`
        UPDATE chat_history
        SET updated_at =  strftime('%s', 'now'),
            chat_log = ?
        WHERE id = ?
    `, entry.ChatLog, entry.Id)

	return err
}

var _ ChatHistoryRepo = (*ChatHistorySqliteRepo)(nil)
