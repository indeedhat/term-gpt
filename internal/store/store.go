package store

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// Connect to the sqlite database
// This will create the database file if it does not exist
func Connect() (*sql.DB, error) {
	return sql.Open("sqlite3", "chatLog.db")
}

// AutoMigrate will run the migration method on sqlite repos in turn
func AutoMigrate(db *sql.DB) error {
	r := NewChatHistorySqliteRepo(db)
	return r.MigrateSchema()
}

// substr is a utf8 safe substring extractor function that respects string length
func substr(input string, start int, length int) string {
	runes := []rune(input)

	if start >= len(runes) {
		return ""
	}

	if start+length > len(runes) {
		length = len(runes) - start
	}

	return string(runes[start : start+length])
}
