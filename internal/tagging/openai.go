package tagging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type OpenaiTagger struct {
	cl  *openai.Client
	key string
}

func NewTagger(openapiApiKey string) OpenaiTagger {
	client := openai.NewClient(openapiApiKey)
	return OpenaiTagger{cl: client, key: openapiApiKey}
}

func (tagger OpenaiTagger) GuessTags(title, content string) ([]string, error) {
	if tagger.key == "" {
		return nil, errors.New("key was not provided for tagging system")
	}
	if content == "" {
		return nil, errors.New("no content -> no tags")
	}
	// cut content till 1000 symbols
	cut := len(content)
	if cut > 1024 {
		cut = 1024
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

	return tags, nil
}

const taggingPrompt = `
Your task is to analyze the title and content of a given article, primarily from the fields of technology, management, or science, and generate a valid JSON array containing three hierarchical tags. Follow these guidelines for the tags:
1. The first tag pinpoints the article's most specific aspect.
2. The second tag broadens the scope slightly.
3. The third tag denotes the widest category that encompasses the article. It must be one of the following phrases: programming, software engineering, infrastructure, management, science, business, productivity, fun.
4. Avoid using words directly from the article's title or content, except for those universally recognized within the domain or part of established ontologies.
4. All tags must be in English, even for articles originally in Russian or any other language.

Ensure the output is a valid JSON array of strings, not an array of objects. Each string is a single tag.
Example Correct Response Format:
["performance", "measurements", "technology"]
`
