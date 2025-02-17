package tagging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	openai "github.com/sashabaranov/go-openai"
)

type OpenaiTagger struct {
	cl  *openai.Client
	key string
}

func NewOpenaiTagger(openapiApiKey string, proxyUrl *url.URL) OpenaiTagger {
	config := openai.DefaultConfig(openapiApiKey)

	if proxyUrl != nil {
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		}
		config.HTTPClient = &http.Client{
			Transport: transport,
		}
	}

	client := openai.NewClientWithConfig(config)
	return OpenaiTagger{cl: client, key: openapiApiKey}
}

func (tagger OpenaiTagger) GuessTags(title, content string) ([]string, error) {
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
	resp, err := tagger.cl.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: taggingPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Title: %s", title),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Content of article: %s", content[:cut]),
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("OpenAI ChatCompletion error: %v\n", err)
		return nil, err
	}

	dataJson := resp.Choices[0].Message.Content
	fmt.Printf("OpenAI ChatCompletion response: %s\n", dataJson)
	var tags []string
	err = json.Unmarshal([]byte(dataJson), &tags)

	if err != nil {
		return nil, err
	}

	tags = append(tags, "autotag")
	return tags, nil
}
