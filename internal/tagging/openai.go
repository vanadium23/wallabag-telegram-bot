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
Your task is to analyze the title and content of a given article and generate a valid JSON array containing with three hierarchical values. Follow these guidelines for the values:
1. All values must be in English, even for articles originally in Russian or any other language.
2. All values must be in lowercase.
3. All values must be one or two words long.
4. One value must be from the following list: programming, software engineering, infrastructure, management, science, business, productivity, fun.
5. One value must be from the following list: python, golang, django, rust, javascript, bash, testing, refactoring, software architecture, system design, microservices, api, event driven architecture, monolith, cloud, docker, nginx, k8s, database, postgresql, shell, monitoring, devops, product management, project management, communication, documentation, leadership, agile, estimates, practices, hiring, decision making, system thinking, history, physics, quantum computing, career, startup, finance, time management, note taking, writing, obsidian, learning, videogame, boardgame, book, rant.
6. Avoid using words directly from the article's title or content, except for those universally recognized within the domain or part of established ontologies.

Ensure the output is a valid JSON array of strings, not an array of objects.
Example Correct Response Format:
["software engineering", "system design", "python"]
`
