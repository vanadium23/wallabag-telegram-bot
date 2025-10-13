package tagging

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/vanadium23/wallabag-telegram-bot/pkg/chatclient"
)

type LLMTagger struct {
	cl    *chatclient.Client
	key   string
	model string
}

// NewLLMTagger creates a new tagger with the specified baseURL, API key, and model
func NewLLMTagger(baseURL, apiKey, model string, proxyUrl *url.URL) LLMTagger {
	client := chatclient.NewClient(baseURL, apiKey, proxyUrl)
	return LLMTagger{
		cl:    client,
		key:   apiKey,
		model: model,
	}
}

func (tagger LLMTagger) GuessTags(title, content string) ([]string, error) {
	if tagger.key == "" {
		return nil, errors.New("key was not provided for tagging system")
	}
	if content == "" {
		return nil, errors.New("no content -> no tags")
	}
	cut := len(content)
	if cut > 4096 {
		cut = 4096
	}

	resp, err := tagger.cl.CreateChatCompletion(chatclient.ChatCompletionRequest{
		Model: tagger.model,
		Messages: []chatclient.Message{
			{
				Role:    "system",
				Content: taggingPrompt,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("Title: %s", title),
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("Content of article: %s", content[:cut]),
			},
		},
	})

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return nil, err
	}

	dataJson := resp.Choices[0].Message.Content
	fmt.Printf("ChatCompletion response: %s\n", dataJson)
	var tags []string
	err = json.Unmarshal([]byte(dataJson), &tags)

	if err != nil {
		return nil, err
	}

	tags = append(tags, "autotag")
	return tags, nil
}
