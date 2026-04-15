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

	"github.com/Korp_Teste_PedroFrosi/billing/models"
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

type AnalysisService struct {
	client *http.Client
	apiKey string
	model  string
}

func NewAnalysisService() *AnalysisService {
	model := os.Getenv("GROQ_MODEL")
	if model == "" {
		model = "llama-3.1-8b-instant"
	}

	return &AnalysisService{
		client: &http.Client{Timeout: 15 * time.Second},
		apiKey: os.Getenv("GROQ_API_KEY"),
		model:  model,
	}
}

func (s *AnalysisService) Analyze(context string, items []models.AIAnalysisItem) (*models.AIAnalysisResponse, error) {
	if strings.TrimSpace(s.apiKey) == "" {
		return nil, fmt.Errorf("ai_analysis: GROQ_API_KEY not configured")
	}

	payload := models.AIAnalysisRequest{
		Context: context,
		Items:   items,
	}

	analysisJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("ai_analysis: marshal request payload: %w", err)
	}

	prompt := fmt.Sprintf(
		`Você é um assistente fiscal para notas de compra.
	Analise a nota abaixo e retorne SOMENTE um JSON válido, sem markdown, sem texto extra.
	
	Regras do JSON:
	- summary: string curta em pt-BR resumindo a nota.
	- category: string com a categoria principal sugerida.
	- risk_level: uma destas strings: "baixo", "medio" ou "alto".
	- alerts: array de strings com alertas ou inconsistências encontradas.
	- recommendations: array de strings com recomendações práticas.
	
	Não há preço, valor total, custo unitário ou orçamento no JSON. Nunca use essas ideias para classificar a nota.
	Não marque risco alto apenas porque a nota mistura notebooks e smartphones. Isso pode ser uma compra normal.
	Use risco alto somente quando houver inconsistências claras, descrições muito genéricas ou forte falta de contexto.
	Se não houver alerta real, use um array vazio.
	Se não houver recomendação, use um array vazio.
	Não altere os itens. Não sugira produtos. Apenas analise coerência e qualidade do cadastro.
	
	Nota em JSON:
	%s`, analysisJSON,
	)

	body, err := json.Marshal(groqRequest{
		Model:       s.model,
		Temperature: 0.2,
		Messages: []groqMessage{
			{Role: "system", Content: "Retorne somente um JSON válido e compacto, sem markdown e sem explicações."},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("ai_analysis: marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, groqAPI, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ai_analysis: build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai_analysis: http call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai_analysis: api error %d: %s", resp.StatusCode, string(raw))
	}

	var gr groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, fmt.Errorf("ai_analysis: decode response: %w", err)
	}

	if len(gr.Choices) == 0 {
		return nil, fmt.Errorf("ai_analysis: empty response from model")
	}

	content := strings.TrimSpace(gr.Choices[0].Message.Content)
	if content == "" {
		return nil, fmt.Errorf("ai_analysis: empty content from model")
	}

	var analysis models.AIAnalysisResponse
	if err := json.Unmarshal([]byte(content), &analysis); err != nil {
		content = sanitizeJSONObject(content)
		if err := json.Unmarshal([]byte(content), &analysis); err != nil {
			return nil, fmt.Errorf("ai_analysis: parse analysis: %w; content=%q", err, content)
		}
	}

	normalizeAnalysisResponse(&analysis)
	applyGuardrails(&analysis, context, items)
	return &analysis, nil
}

func sanitizeJSONObject(content string) string {
	trimmed := strings.TrimSpace(content)

	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		return trimmed[start : end+1]
	}

	return trimmed
}

func normalizeAnalysisResponse(analysis *models.AIAnalysisResponse) {
	if analysis == nil {
		return
	}

	analysis.Summary = strings.TrimSpace(analysis.Summary)
	analysis.Category = strings.TrimSpace(analysis.Category)
	analysis.RiskLevel = strings.ToLower(strings.TrimSpace(analysis.RiskLevel))

	for index, alert := range analysis.Alerts {
		analysis.Alerts[index] = strings.TrimSpace(alert)
	}

	for index, recommendation := range analysis.Recommendations {
		analysis.Recommendations[index] = strings.TrimSpace(recommendation)
	}
}

func applyGuardrails(analysis *models.AIAnalysisResponse, context string, items []models.AIAnalysisItem) {
	if analysis == nil {
		return
	}

	analysis.Summary = sanitizeNarrative(analysis.Summary)
	analysis.Category = sanitizeNarrative(analysis.Category)
	analysis.Recommendations = filterValueBasedLines(analysis.Recommendations)
	analysis.Alerts = mergeUniqueLines(
		filterValueBasedLines(analysis.Alerts),
		buildDeterministicAlerts(context, items)...,
	)
	analysis.RiskLevel = determineRiskLevel(analysis.Alerts)

	if analysis.Summary == "" {
		analysis.Summary = "Análise concluída com base nos itens informados."
	}
	if analysis.Category == "" {
		analysis.Category = "Análise da nota fiscal"
	}
}

func sanitizeNarrative(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	if containsMonetaryClaim(trimmed) {
		return "Análise concluída com base na consistência dos itens informados."
	}

	return trimmed
}

func filterValueBasedLines(lines []string) []string {
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if containsMonetaryClaim(trimmed) {
			continue
		}
		filtered = append(filtered, trimmed)
	}

	return filtered
}

func mergeUniqueLines(base []string, extra ...string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	result := make([]string, 0, len(base)+len(extra))

	appendLine := func(line string) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}

	for _, line := range base {
		appendLine(line)
	}
	for _, line := range extra {
		appendLine(line)
	}

	return result
}

func buildDeterministicAlerts(context string, items []models.AIAnalysisItem) []string {
	alerts := make([]string, 0)
	for index, item := range items {
		if isGenericDescription(item.Description) {
			label := item.ProductCode
			if strings.TrimSpace(label) == "" {
				label = fmt.Sprintf("item %d", index+1)
			}
			alerts = append(alerts, fmt.Sprintf("Descrição pouco específica no %s. Revise o cadastro.", label))
		}
	}

	if hasMultipleProductFamilies(items) && !isHelpfulContext(context) {
		alerts = append(alerts, "Itens de categorias diferentes no mesmo rascunho; confirme se essa composição está correta.")
	}

	return alerts
}

func determineRiskLevel(alerts []string) string {
	if len(alerts) == 0 {
		return "baixo"
	}

	genericCount := 0
	mixedCount := 0
	for _, alert := range alerts {
		lower := strings.ToLower(alert)
		if strings.Contains(lower, "descrição pouco específica") {
			genericCount++
		}
		if strings.Contains(lower, "categorias diferentes") {
			mixedCount++
		}
	}

	if genericCount >= 2 || (genericCount >= 1 && mixedCount >= 1) {
		return "alto"
	}

	return "medio"
}

func containsMonetaryClaim(text string) bool {
	lower := strings.ToLower(text)
	for _, term := range []string{
		"valor alto",
		"alto valor",
		"preço",
		"preco",
		"custo",
		"caro",
		"barato",
		"orçamento",
		"orcamento",
	} {
		if strings.Contains(lower, term) {
			return true
		}
	}

	return false
}

func isGenericDescription(description string) bool {
	normalized := strings.ToLower(strings.TrimSpace(description))
	if normalized == "" {
		return true
	}

	if len([]rune(normalized)) < 8 {
		return true
	}

	for _, term := range []string{
		"produto",
		"material",
		"materiais",
		"diverso",
		"diversos",
		"vários",
		"varios",
		"outro",
		"outros",
		"item",
		"itens",
		"mercadoria",
		"mercadorias",
		"compra",
	} {
		if normalized == term || strings.Contains(normalized, term) {
			return true
		}
	}

	return false
}

func hasMultipleProductFamilies(items []models.AIAnalysisItem) bool {
	families := make(map[string]struct{})
	for _, item := range items {
		family := detectProductFamily(item.Description)
		if family == "" {
			continue
		}
		families[family] = struct{}{}
	}

	return len(families) > 1
}

func detectProductFamily(description string) string {
	lower := strings.ToLower(description)

	switch {
	case containsAny(lower, "notebook", "laptop", "computador", "pc", "macbook"):
		return "computing"
	case containsAny(lower, "iphone", "smartphone", "celular", "android", "tablet", "ipad"):
		return "mobile"
	case containsAny(lower, "monitor", "tela", "display"):
		return "display"
	case containsAny(lower, "impressora", "printer"):
		return "printer"
	case containsAny(lower, "mouse", "teclado", "keyboard", "fone", "headset"):
		return "accessory"
	case containsAny(lower, "papel", "caderno", "caneta", "lápis", "lapis"):
		return "stationery"
	case containsAny(lower, "cadeira", "mesa", "móvel", "movel"):
		return "furniture"
	default:
		return ""
	}
}

func containsAny(text string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}

	return false
}

func isHelpfulContext(context string) bool {
	return len(strings.Fields(strings.TrimSpace(context))) >= 2
}