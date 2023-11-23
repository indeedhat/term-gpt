package store

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/sashabaranov/go-openai"
)

type ChatLog []openai.ChatCompletionMessage

// Value implements driver.Valuer.
func (l *ChatLog) Value() (driver.Value, error) {
	if l == nil {
		return nil, nil
	}

	data, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}

	return data, nil
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
	ChatHistoryMini

	ChatLog
}

type ChatHistoryMini struct {
	Id        int
	Title     string
	UpdatedAt time.Time
}

type ChatHistoryRepo interface {
	Create() error
	Update() error
	List() []ChatHistoryMini
	Find(id int) ChatHistory
}
