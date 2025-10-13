package chatclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client is a simple OpenAI-compatible chat completion client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents the request to chat/completions endpoint
type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// ChatCompletionResponse represents the response from chat/completions endpoint
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewClient creates a new chat completion client
// baseURL: the base URL of the API (e.g., "https://api.openai.com/v1")
// apiKey: the API key for authentication
// proxyURL: optional proxy URL (can be nil)
func NewClient(baseURL, apiKey string, proxyURL *url.URL) *Client {
	httpClient := &http.Client{}
	if proxyURL != nil {
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		httpClient.Transport = transport
	}

	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// CreateChatCompletion sends a chat completion request and returns the structured response
func (c *Client) CreateChatCompletion(req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	endpoint := fmt.Sprintf("%s/chat/completions", c.baseURL)
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("API error: %s", errResp.Error.Message)
	}

	// Parse successful response
	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &chatResp, nil
}
