package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const groqAPI = "https://api.groq.com/openai/v1/chat/completions"

type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type SuggestionService struct {
	client *http.Client
	apiKey string
}

func NewSuggestionService() *SuggestionService {
	return &SuggestionService{
		client: &http.Client{},
		apiKey: os.Getenv("GROQ_API_KEY"),
	}
}

func (s *SuggestionService) Suggest(description string) ([]string, error) {
	prompt := fmt.Sprintf(
		`You are a product catalog assistant. Given the invoice description below, suggest up to 5 related product names that could be added to this invoice. Return ONLY a JSON array of strings, no explanation.

Description: %s`, description,
	)

	body, err := json.Marshal(groqRequest{
		Model: "llama3-8b-8192",
		Messages: []groqMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("ai_suggestion: marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, groqAPI, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ai_suggestion: build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai_suggestion: http call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai_suggestion: api error %d: %s", resp.StatusCode, string(raw))
	}

	var gr groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, fmt.Errorf("ai_suggestion: decode response: %w", err)
	}

	if len(gr.Choices) == 0 {
		return nil, fmt.Errorf("ai_suggestion: empty response from model")
	}

	var suggestions []string
	if err := json.Unmarshal([]byte(gr.Choices[0].Message.Content), &suggestions); err != nil {
		return nil, fmt.Errorf("ai_suggestion: parse suggestions: %w", err)
	}

	return suggestions, nil
}