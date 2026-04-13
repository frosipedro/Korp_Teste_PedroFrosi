package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const anthropicAPI = "https://api.anthropic.com/v1/messages"

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

type SuggestionService struct {
	client *http.Client
	apiKey string
}

func NewSuggestionService() *SuggestionService {
	return &SuggestionService{
		client: &http.Client{},
		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
	}
}

func (s *SuggestionService) Suggest(description string) ([]string, error) {
	prompt := fmt.Sprintf(
		`You are a product catalog assistant. Given the invoice description below, suggest up to 5 related product names that could be added to this invoice. Return ONLY a JSON array of strings, no explanation.

Description: %s`, description,
	)

	body, err := json.Marshal(anthropicRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 256,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("ai_suggestion: marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, anthropicAPI, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ai_suggestion: build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai_suggestion: http call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai_suggestion: api error %d: %s", resp.StatusCode, string(raw))
	}

	var ar anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("ai_suggestion: decode response: %w", err)
	}

	if len(ar.Content) == 0 {
		return nil, fmt.Errorf("ai_suggestion: empty response from model")
	}

	var suggestions []string
	if err := json.Unmarshal([]byte(ar.Content[0].Text), &suggestions); err != nil {
		return nil, fmt.Errorf("ai_suggestion: parse suggestions: %w", err)
	}

	return suggestions, nil
}