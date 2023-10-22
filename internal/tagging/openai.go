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
Generate a JSON array containing three hierarchical tags for a given article, primarily from the fields of technology, management, or science. Each tag should preferably be a lowercase English noun consisting of one word, though two-word noun phrases are acceptable. The tags should adhere to the following criteria:
1. The first tag pinpoints the article's most specific aspect.
2. The second tag broadens the scope slightly.
3. The third tag denotes the widest category that encompasses the article.
4. Tags must not replicate words from the article's title or content, except for words universally recognized within the domain or established ontologies.
5. Regardless of the article's original language, all tags should be in English.

Title: %s

Content:
%s
`
