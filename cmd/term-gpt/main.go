package main

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indeedhat/term-gpt/internal/gpt"
	"github.com/indeedhat/term-gpt/internal/store"

	"github.com/indeedhat/term-gpt/internal/env"
	"github.com/sashabaranov/go-openai"
)

func main() {
	conf := openai.DefaultConfig(env.Get(env.OpenAiToken))
	if org := env.Get(env.OpenAiOrg); org != "" {
		conf.OrgID = env.Get(env.OpenAiOrg)
	}
	client := openai.NewClientWithConfig(conf)

	db, err := store.Connect()
	if err != nil {
		log.Fatal(err)
	}

	repo := store.NewChatHistorySqliteRepo(db)
	if err := repo.MigrateSchema(); err != nil {
		log.Fatal(err)
	}

	prog := tea.NewProgram(gpt.New(repo, client), tea.WithAltScreen())

	// horrible hack
	go func() {
		time.Sleep(50 * time.Millisecond)
		prog.Send(prog)
	}()

	if _, err := prog.Run(); err != nil {
		log.Fatal(err)
	}
}
