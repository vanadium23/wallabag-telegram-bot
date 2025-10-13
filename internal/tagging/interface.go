package tagging

import (
	_ "embed"
	"net/url"
)

type Tagger interface {
	GuessTags(title, content string) ([]string, error)
}

//go:embed tagging_prompt.txt
var taggingPrompt string

func NewTagger(
	openaiApiKey string, proxyUrl *url.URL,
	openrouterApiKey string, openrouterModel string,
) Tagger {
	if openrouterApiKey != "" {
		if openrouterModel == "" {
			openrouterModel = "anthropic/claude-3.5-sonnet"
		}
		return NewLLMTagger("https://openrouter.ai/api/v1", openrouterApiKey, openrouterModel, proxyUrl)
	}
	return NewLLMTagger("https://api.openai.com/v1", openaiApiKey, "gpt-4", proxyUrl)
}
