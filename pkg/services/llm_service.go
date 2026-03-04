package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Message represents a chat message for the LLM
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMService handles interactions with the z.ai API
type LLMService struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewLLMService creates a new LLMService
func NewLLMService(apiKey, baseURL string) *LLMService {
	return &LLMService{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   "glm-coding-pro",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// SetModel sets the model to use
func (s *LLMService) SetModel(model string) {
	s.model = model
}

// ChatCompletion performs a chat completion request
func (s *LLMService) ChatCompletion(messages []Message) (string, error) {
	req := ChatRequest{
		Model:    s.model,
		Messages: messages,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", s.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// GenerateScript generates a video script from parsed content
func (s *LLMService) GenerateScript(title, explanation, code string) (string, error) {
	systemPrompt := `You are an expert YouTube Short script writer for programming content in Russian.
Create engaging, conversational scripts for under 60-second videos.
Use "ты" form (friendly, informal).
Focus on the key insight/aha moment.
Keep sentences short and punchy.
Maximum 350 characters.`

	userPrompt := fmt.Sprintf(`Create a 45-50 second Russian voice-over script for a YouTube Short about this Go secret.

Title: %s

Explanation: %s

Code example: %s

Requirements:
- Natural, conversational Russian (like Alice from Yandex)
- Use "ты" form (friendly, informal)
- Keep sentences short and punchy
- Max 350 characters (to fit within TTS limit)
- Focus on the key insight/aha moment

Return only the script text, no explanations.`, title, explanation, code)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	return s.ChatCompletion(messages)
}

// GenerateScriptSimplified generates a simpler script without code
func (s *LLMService) GenerateScriptSimplified(title, explanation string) (string, error) {
	systemPrompt := `You are an expert YouTube Short script writer for programming content in Russian.
Create engaging, conversational scripts for under 60-second videos.
Use "ты" form (friendly, informal).
Focus on the key insight/aha moment.
Keep sentences short and punchy.
Maximum 350 characters.`

	userPrompt := fmt.Sprintf(`Create a 45-50 second Russian voice-over script for a YouTube Short about this Go secret.

Title: %s

Explanation: %s

Requirements:
- Natural, conversational Russian (like Alice from Yandex)
- Use "ты" form (friendly, informal)
- Keep sentences short and punchy
- Max 350 characters (to fit within TTS limit)
- Focus on the key insight/aha moment

Return only the script text, no explanations.`, title, explanation)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	return s.ChatCompletion(messages)
}
