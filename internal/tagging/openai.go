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
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(promptFormat, title, content[:cut]),
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return nil, err
	}

	dataJson := resp.Choices[0].Message.Content
	var tags []string
	err = json.Unmarshal([]byte(dataJson), &tags)

	if err != nil {
		return nil, err
	}

	return tags, nil
}

const promptFormat = `
You response need to contain only valid json array.
Generate three hierarchical tags based on the title and content of a given article, primarily in the fields of technology, management, or science. Each tag should preferably be a lowercase English noun consisting of one word, although two-word noun phrases are acceptable. The first tag should capture the most specific aspect of the article, the second tag should be slightly more general, and the third tag should encapsulate the broadest category the article falls under. 
Articles in Russian should also be tagged in English.

Title: %s

Content:
%s
`
