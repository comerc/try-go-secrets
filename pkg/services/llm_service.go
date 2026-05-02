package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go-secrets-pipeline/pkg/config"
)

type LLMService struct {
	cfg    *config.Config
	client *http.Client
}

type ZAIRequest struct {
	Model    string       `json:"model"`
	Messages []ZAIMessage `json:"messages"`
	Thinking *thinkingCfg `json:"thinking,omitempty"`
}

type thinkingCfg struct {
	Type          string `json:"type"`
	ClearThinking *bool  `json:"clear_thinking,omitempty"`
}

type ZAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ZAIResponse struct {
	Choices []struct {
		Message ZAIMessage `json:"message"`
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
	switch s.cfg.LLMBackend {
	case "codex-cli":
		return s.completeCodex(systemPrompt, userPrompt)
	case "claude-cli":
		return s.completeCLI(systemPrompt, userPrompt)
	case "zai-api":
		return s.completeZAI(systemPrompt, userPrompt)
	default:
		return "", fmt.Errorf("неизвестный LLM_BACKEND: %s", s.cfg.LLMBackend)
	}
}

func (s *LLMService) completeCLI(systemPrompt, userPrompt string) (string, error) {
	combined := systemPrompt + "\n\n---\n\n" + userPrompt

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	effort := s.cfg.LLMEffort
	if effort == "" {
		effort = "medium"
	}

	args := []string{"-p", combined, "--output-format", "text"}
	if s.cfg.LLMModel != "" {
		args = append(args, "--model", s.cfg.LLMModel)
	}
	args = append(args, "--effort", effort)
	cmd := exec.CommandContext(ctx, "claude", args...)
	out, err := cmd.Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude-cli: %w\nstderr: %s", err, e.Stderr)
		}
		return "", fmt.Errorf("claude-cli: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

func (s *LLMService) completeCodex(systemPrompt, userPrompt string) (string, error) {
	combined := systemPrompt + "\n\n---\n\n" + userPrompt

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "go-secrets-codex-*")
	if err != nil {
		return "", fmt.Errorf("codex-cli: создать temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	effort := s.cfg.LLMEffort
	if effort == "" {
		effort = "medium"
	}

	lastMessagePath := filepath.Join(tmpDir, "last-message.txt")
	args := []string{"exec", "--ephemeral", "--output-last-message", lastMessagePath, "--cd", "."}
	if s.cfg.LLMModel != "" {
		args = append(args, "--model", s.cfg.LLMModel)
	}
	args = append(args, "--config", "model_reasoning_effort="+effort)
	args = append(args, "-")
	cmd := exec.CommandContext(ctx, "codex", args...)
	cmd.Stdin = strings.NewReader(combined)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("codex-cli: %w\noutput: %s", err, string(out))
	}

	msg, err := os.ReadFile(lastMessagePath)
	if err != nil {
		return "", fmt.Errorf("codex-cli: прочитать ответ: %w\noutput: %s", err, string(out))
	}

	return strings.TrimSpace(string(msg)), nil
}

func (s *LLMService) completeZAI(systemPrompt, userPrompt string) (string, error) {
	model := s.cfg.LLMModel
	if model == "" {
		model = "glm-5.1"
	}
	effort := s.cfg.LLMEffort
	if effort == "" {
		effort = "medium"
	}

	thinkingEnabled := effort != "low"
	preservedThinking := effort == "high"

	reqBody := ZAIRequest{
		Model: model,
		Messages: []ZAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Thinking: &thinkingCfg{Type: "disabled"},
	}
	if thinkingEnabled {
		reqBody.Thinking.Type = "enabled"
	}
	if preservedThinking {
		clearThinking := false
		reqBody.Thinking.ClearThinking = &clearThinking
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

	var llmResp ZAIResponse
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
