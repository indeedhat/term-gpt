package gpt

import (
	"fmt"

	"github.com/indeedhat/term-gpt/internal/env"
	"github.com/indeedhat/term-gpt/internal/store"
	"github.com/sashabaranov/go-openai"
)

// sendGptRequest sends off a completion request to the chat gpt api
func sendGptRequest(m *Model) {
	m.program.Send(spinMsg(true))

	var (
		msgs     store.ChatLog
		msgCount = len(m.activeChat.history.ChatLog)
		maxMsgs  = env.GetInt(env.MaxPrevMesgs)
	)

	if maxMsgs == 0 || msgCount <= maxMsgs {
		msgs = m.activeChat.history.ChatLog
	} else {
		msgs = m.activeChat.history.ChatLog[msgCount-maxMsgs:]
	}

	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		Messages:  msgs,
		MaxTokens: env.GetInt(env.MaxRequestTokens),
	}
	resp, err := m.client.CreateChatCompletion(m.ctx, req)

	if err != nil {
		fmt.Print(err)
		m.program.Send(chatResultMsg{err: err})
		return
	}

	m.program.Send(chatResultMsg{message: resp.Choices[0].Message.Content})
}
