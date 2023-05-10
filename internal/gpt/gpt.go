package gpt

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"gitlab.com/milad.arab2010/dubai-backend/internal/config"

	"github.com/rs/zerolog/log"
	ai "github.com/sashabaranov/go-openai"
)

type Client struct {
	client     *ai.Client
	history    []ai.ChatCompletionMessage
	model      string
	apiUrl     string
	apiTimeout time.Duration
}

type Chunk struct {
	Content string
	Err     error
}

func NewClient(cnf config.ChatGPT) (*Client, error) {
	client := &Client{
		client: ai.NewClient(cnf.ApiKey),
		// Typically, a conversation is formatted with a system message first,
		// followed by alternating user and assistant messages.
		// Ref: https://platform.openai.com/docs/guides/chat/introduction
		history:    []ai.ChatCompletionMessage{},
		model:      cnf.Model,
		apiUrl:     cnf.ApiUrl,
		apiTimeout: cnf.ApiTimeoutSecond,
	}

	if err := client.instruct(cnf); err != nil {
		return nil, err
	}

	return client, nil
}

// instruct the model
func (b *Client) instruct(cnf config.ChatGPT) error {
	var instruct string
	if len(cnf.InstructionFilePath) > 0 {
		file, err := ioutil.ReadFile(cnf.InstructionFilePath)
		if err != nil {
			return err
		}
		instruct = string(file)
	}

	if len(cnf.InstructionText) > 0 {
		instruct = cnf.InstructionText
	}

	b.history = append(b.history, ai.ChatCompletionMessage{
		Role:    ai.ChatMessageRoleSystem,
		Content: instruct,
	})

	return nil
}

func (b *Client) Prompt(question string) (<-chan Chunk, error) {

	req := b.newChatCompletionRequest(question)
	ctx, cancel := context.WithTimeout(context.Background(), b.apiTimeout)
	resp, err := b.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		cancel()
		log.Error().Err(err).Msg("failed to create chat completion stream")
		return nil, err
	}

	c := make(chan Chunk)

	timeoutDuration := time.Second * 5
	timeout := time.NewTimer(timeoutDuration)

	writeOnCh := func(chk Chunk) {
		timeout.Reset(timeoutDuration)
		select {
		case c <- chk:
			return
		case <-timeout.C: // dont block writter
			return
		}
	}

	go func() {
		defer cancel()
		defer close(c)
		defer resp.Close()
		defer timeout.Stop()

		sb := strings.Builder{}
		for {
			var data ai.ChatCompletionStreamResponse
			data, err = resp.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					log.Error().Err(err).Msg("stream error")
				}
				writeOnCh(Chunk{
					Content: "",
					Err:     err,
				})
				break
			}
			respChunk := data.Choices[0].Delta.Content
			sb.WriteString(respChunk)
			writeOnCh(Chunk{
				Content: respChunk,
				Err:     nil,
			})
		}

		b.history = append(b.history, ai.ChatCompletionMessage{
			Role:    ai.ChatMessageRoleAssistant,
			Content: sb.String(),
		})
	}()

	return c, nil
}

func (b *Client) newChatCompletionRequest(question string) ai.ChatCompletionRequest {

	/*
		Ref: https://platform.openai.com/docs/guides/chat/introduction
		Including the conversation history helps the models to give relevant answers to the prior conversation.
		Because the models have no memory of past requests, all relevant information must be supplied via the conversation.
	*/
	b.history = append(b.history, ai.ChatCompletionMessage{
		Role:    ai.ChatMessageRoleUser,
		Content: question,
	})

	return ai.ChatCompletionRequest{
		Model:    b.model,
		Messages: b.history,
		Stream:   true,
	}
}
