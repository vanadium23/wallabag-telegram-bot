package summarization

import "net/url"

type Summarizer interface {
	Summarize(title, content string) (string, error)
}

const summarizationPrompt = `
Your task is to create a concise and informative summary of the given article. Follow these guidelines:
1. The summary should be in English, even if the article is in another language.
2. Keep the summary between 100-200 words.
3. Focus on the main points and key takeaways.
4. Maintain a professional and objective tone.
5. Avoid personal opinions or commentary.
6. Ensure the summary is coherent and well-structured.

The summary should be returned as plain text, without any additional formatting or markdown.
`

func NewSummarizer(
	openaiApiKey string, proxyUrl *url.URL,
	openrouterApiKey string, openrouterModel string,
) Summarizer {
	if openrouterApiKey != "" {
		return NewOpenrouterSummarizer(openrouterApiKey, proxyUrl, openrouterModel)
	}
	return NewOpenaiSummarizer(openaiApiKey, proxyUrl)
}
