package tagging

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	openrouterapigo "github.com/wojtess/openrouter-api-go"
)

type OpenrouterTagger struct {
	cl    *openrouterapigo.OpenRouterClient
	key   string
	model string
}

func NewOpenrouterTagger(openrouterApiKey string, proxyUrl *url.URL, model string) OpenrouterTagger {
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
	return OpenrouterTagger{cl: client, key: openrouterApiKey, model: model}
}

func (tagger OpenrouterTagger) GuessTags(title, content string) ([]string, error) {
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

	request := openrouterapigo.Request{
		Model: tagger.model,
		Messages: []openrouterapigo.MessageRequest{
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
	}

	resp, err := tagger.cl.FetchChatCompletions(request)

	if err != nil {
		fmt.Printf("Openrouter ChatCompletion error: %v\n", err)
		return nil, err
	}

	dataJson := resp.Choices[0].Message.Content
	fmt.Printf("Openrouter ChatCompletion response: %s\n", dataJson)
	var tags []string
	err = json.Unmarshal([]byte(dataJson), &tags)

	if err != nil {
		return nil, err
	}

	tags = append(tags, "autotag")
	return tags, nil
}
