package tagging

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

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
	// Check if content is empty or contains the wallabag error message
	if strings.TrimSpace(content) == "" || strings.Contains(strings.ToLower(content), "wallabag can't retrieve contents for this article") {
		return []string{"autotag:skipped"}, nil
	}

	// Calculate content limit based on available tokens
	// Approximation: 1 token ≈ 4 characters
	// System prompt (~875 tokens) + Title (~200 tokens) + Response (~200 tokens) + Buffer (~500 tokens)
	// = ~1775 tokens overhead = ~7100 characters
	// For 100k tokens total context: ~400k characters available
	// Conservative limit: 100k characters for content
	const maxContentChars = 100000
	cut := len(content)
	if cut > maxContentChars {
		cut = maxContentChars
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

	// Parse as map[string][]string to get categorized tags
	var tagData map[string][]string
	err = json.Unmarshal([]byte(dataJson), &tagData)

	if err != nil {
		return nil, err
	}

	// Convert map to formatted tags (category:item)
	var tags []string
	for key, values := range tagData {
		for _, item := range values {
			tags = append(tags, fmt.Sprintf("%s:%s", key, item))
		}
	}

	tags = append(tags, "autotag:v2")
	return tags, nil
}
