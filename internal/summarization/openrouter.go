package summarization

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	openrouterapigo "github.com/wojtess/openrouter-api-go"
)

type OpenrouterSummarizer struct {
	cl    *openrouterapigo.OpenRouterClient
	key   string
	model string
}

func NewOpenrouterSummarizer(openrouterApiKey string, proxyUrl *url.URL, model string) OpenrouterSummarizer {
	httpClient := &http.Client{}
	if proxyUrl != nil {
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		}
		httpClient.Transport = transport
	}

	client := openrouterapigo.NewOpenRouterClientFull(
		strings.Trim(openrouterApiKey, " "), "https://openrouter.ai/api/v1",
		httpClient,
	)
	if model == "" {
		model = "anthropic/claude-3.5-sonnet"
	}
	return OpenrouterSummarizer{cl: client, key: openrouterApiKey, model: model}
}

func (summarizer OpenrouterSummarizer) Summarize(title, content string) (string, error) {
	if summarizer.key == "" {
		return "", fmt.Errorf("key was not provided for summarization system")
	}
	if content == "" {
		return "", fmt.Errorf("no content -> no summary")
	}

	request := openrouterapigo.Request{
		Model: summarizer.model,
		Messages: []openrouterapigo.MessageRequest{
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
				Content: fmt.Sprintf("Content of article: %s", content),
			},
		},
	}

	resp, err := summarizer.cl.FetchChatCompletions(request)
	if err != nil {
		fmt.Printf("Openrouter ChatCompletion error: %v\n", err)
		return "", err
	}

	summary := resp.Choices[0].Message.Content
	fmt.Printf("Openrouter ChatCompletion response: %s\n", summary)
	return summary, nil
}
