package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"go-secrets-pipeline/pkg/config"
)

type LLMService struct {
	cfg    *config.Config
	client *http.Client
}

type llmRequest struct {
	Model    string       `json:"model"`
	Messages []llmMessage `json:"messages"`
}

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmResponse struct {
	Choices []struct {
		Message llmMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewLLMService(cfg *config.Config) *LLMService {
	return &LLMService{
		cfg:    cfg,
		client: &http.Client{Timeout: 5 * time.Minute},
	}
}

// Complete отправляет запрос к LLM-бэкенду и возвращает ответ
func (s *LLMService) Complete(systemPrompt, userPrompt string) (string, error) {
	if s.cfg.LLMBackend == "claude-cli" {
		return s.completeCLI(systemPrompt, userPrompt)
	}
	return s.completeZAI(systemPrompt, userPrompt)
}

func (s *LLMService) completeCLI(systemPrompt, userPrompt string) (string, error) {
	combined := systemPrompt + "\n\n---\n\n" + userPrompt

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "-p", combined, "--output-format", "text")
	out, err := cmd.Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude-cli: %w\nstderr: %s", err, e.Stderr)
		}
		return "", fmt.Errorf("claude-cli: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

func (s *LLMService) completeZAI(systemPrompt, userPrompt string) (string, error) {
	reqBody := llmRequest{
		Model: s.cfg.ZAIModel,
		Messages: []llmMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", s.cfg.ZAIApiURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.ZAIApiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("z.ai запрос: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("z.ai ошибка %d: %s", resp.StatusCode, string(body))
	}

	var llmResp llmResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		return "", fmt.Errorf("парсинг z.ai ответа: %w", err)
	}

	if llmResp.Error != nil {
		return "", fmt.Errorf("z.ai API error: %s", llmResp.Error.Message)
	}

	if len(llmResp.Choices) == 0 {
		return "", fmt.Errorf("z.ai вернул пустой ответ")
	}

	return llmResp.Choices[0].Message.Content, nil
}
