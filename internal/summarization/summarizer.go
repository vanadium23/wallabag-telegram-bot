package summarization

import (
	"fmt"
	"net/url"

	"github.com/vanadium23/wallabag-telegram-bot/pkg/chatclient"
)

type LLMSummarizer struct {
	cl    *chatclient.Client
	key   string
	model string
}

// NewLLMSummarizer creates a new summarizer with the specified baseURL, API key, and model
func NewLLMSummarizer(baseURL, apiKey, model string, proxyUrl *url.URL) LLMSummarizer {
	client := chatclient.NewClient(baseURL, apiKey, proxyUrl)
	return LLMSummarizer{
		cl:    client,
		key:   apiKey,
		model: model,
	}
}

func (summarizer LLMSummarizer) Summarize(title, content string) (string, error) {
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

	resp, err := summarizer.cl.CreateChatCompletion(chatclient.ChatCompletionRequest{
		Model: summarizer.model,
		Messages: []chatclient.Message{
			{
				Role:    "system",
				Content: summarizationPrompt,
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
		return "", err
	}

	summary := resp.Choices[0].Message.Content
	fmt.Printf("ChatCompletion response: %s\n", summary)
	return summary, nil
}
