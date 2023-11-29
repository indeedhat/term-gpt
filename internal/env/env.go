package env

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	OpenAiToken      = "OPEN_AI_TOKEN"
	OpenAiOrg        = "OPEN_AI_ORG"
	MaxRequestTokens = "MAX_REQUEST_TOKENS"
	MaxPrevMesgs     = "MAX_PREV_MSGS"
)

var loaded bool

// Get a value from the environment
func Get(key string) string {
	Load()
	return os.Getenv(key)
}

// GetInt gets a value from the environment as an int
// if the value is not found or is not an int then 0 will be renurned
func GetInt(key string) int {
	Load()
	val := os.Getenv(key)
	if val == "" {
		return 0
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}

	return i
}

// Load the .env file into the environment
func Load() {
	if loaded {
		return
	}

	godotenv.Load()
	loaded = true
}
