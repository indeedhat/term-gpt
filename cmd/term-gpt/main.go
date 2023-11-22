package main

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indeedhat/term-gpt/internal/gpt"

	"github.com/indeedhat/term-gpt/internal/env"
	"github.com/sashabaranov/go-openai"
)

func main() {
	conf := openai.DefaultConfig(env.Get(env.OpenAiToken))
	conf.OrgID = env.Get(env.OpenAiOrg)
	client := openai.NewClientWithConfig(conf)

	prog := tea.NewProgram(gpt.New(client))

	// horrible hack
	go func() {
		time.Sleep(50 * time.Millisecond)
		prog.Send(prog)
	}()

	if _, err := prog.Run(); err != nil {
		log.Fatal(err)
	}
}
