package summarization

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	openai "github.com/sashabaranov/go-openai"
)

type OpenaiSummarizer struct {
	cl  *openai.Client
	key string
}

func NewOpenaiSummarizer(openaiApiKey string, proxyUrl *url.URL) OpenaiSummarizer {
	config := openai.DefaultConfig(openaiApiKey)
	if proxyUrl != nil {
		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			},
		}
	}
	client := openai.NewClientWithConfig(config)
	return OpenaiSummarizer{cl: client, key: openaiApiKey}
}

func (summarizer OpenaiSummarizer) Summarize(title, content string) (string, error) {
	if summarizer.key == "" {
		return "", fmt.Errorf("key was not provided for summarization system")
	}
	if content == "" {
		return "", fmt.Errorf("no content -> no summary")
	}
	cut := len(content)
	if cut > 4096 {
		cut = 4096
	}

	resp, err := summarizer.cl.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: summarizationPrompt,
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
		return "", err
	}

	summary := resp.Choices[0].Message.Content
	fmt.Printf("OpenAI ChatCompletion response: %s\n", summary)
	return summary, nil
}
