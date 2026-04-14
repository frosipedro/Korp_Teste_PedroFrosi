package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const groqAPI = "https://api.groq.com/openai/v1/chat/completions"

type groqRequest struct {
	Model       string        `json:"model"`
	Messages    []groqMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
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
	model  string
}

func NewSuggestionService() *SuggestionService {
	model := os.Getenv("GROQ_MODEL")
	if model == "" {
		model = "llama-3.1-8b-instant"
	}

	return &SuggestionService{
		client: &http.Client{Timeout: 15 * time.Second},
		apiKey: os.Getenv("GROQ_API_KEY"),
		model:  model,
	}
}

func (s *SuggestionService) Suggest(description string) ([]string, error) {
	if strings.TrimSpace(s.apiKey) == "" {
		return nil, fmt.Errorf("ai_suggestion: GROQ_API_KEY not configured")
	}

	prompt := fmt.Sprintf(
		`You are a product catalog assistant. Given the invoice description below, suggest up to 5 related product names that could be added to this invoice. Return ONLY a JSON array of strings, no explanation.

Description: %s`, description,
	)

	body, err := json.Marshal(groqRequest{
		Model:       s.model,
		Temperature: 0.2,
		Messages: []groqMessage{
			{Role: "system", Content: "Return only valid JSON array of strings. No markdown fences, no explanation."},
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

	content := strings.TrimSpace(gr.Choices[0].Message.Content)
	if content == "" {
		return nil, fmt.Errorf("ai_suggestion: empty content from model")
	}

	var suggestions []string
	if err := json.Unmarshal([]byte(content), &suggestions); err != nil {
		// Some models return code fences or extra text; try extracting just the JSON array.
		content = sanitizeJSONArray(content)
		if err := json.Unmarshal([]byte(content), &suggestions); err != nil {
			return nil, fmt.Errorf("ai_suggestion: parse suggestions: %w; content=%q", err, content)
		}
	}

	return suggestions, nil
}

func sanitizeJSONArray(content string) string {
	trimmed := strings.TrimSpace(content)

	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)

	start := strings.Index(trimmed, "[")
	end := strings.LastIndex(trimmed, "]")
	if start >= 0 && end > start {
		return trimmed[start : end+1]
	}

	return trimmed
}
