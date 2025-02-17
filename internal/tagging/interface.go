package tagging

import "net/url"

type Tagger interface {
	GuessTags(title, content string) ([]string, error)
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

func NewTagger(
	openaiApiKey string, proxyUrl *url.URL,
	openrouterApiKey string, openrouterModel string,
) Tagger {
	if openrouterApiKey != "" {
		return NewOpenrouterTagger(openrouterApiKey, proxyUrl, openrouterModel)
	}
	return NewOpenaiTagger(openaiApiKey, proxyUrl)
}
