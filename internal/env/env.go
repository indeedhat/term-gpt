package env

import (
	"os"

	"github.com/joho/godotenv"
)

const (
	OpenAiToken = "OPEN_AI_TOKEN"
	OpenAiOrg   = "OPEN_AI_ORG"
)

var loaded bool

func Get(key string) string {
	Load()
	return os.Getenv(key)
}

func Load() {
	if loaded {
		return
	}

	godotenv.Load()
	loaded = true
}
