package services

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"go-secrets-pipeline/pkg/models"
)

var (
	reCodeBlock = regexp.MustCompile("(?s)```(\\w*)\\n(.*?)```")
	reOldBlock  = regexp.MustCompile("(?s)```old\\n(.*?)```")
)

// ParseMarkdown разбирает markdown-файл с секретом Go
func ParseMarkdown(filePath string) (*models.RawContent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	raw := string(data)
	content := &models.RawContent{
		FilePath: filePath,
		FileNum:  extractFileNum(filePath),
		Title:    extractTitle(raw, filePath),
	}

	// Извлекаем блоки кода (кроме old)
	for _, m := range reCodeBlock.FindAllStringSubmatch(raw, -1) {
		lang := strings.TrimSpace(m[1])
		code := strings.TrimSpace(m[2])
		if lang == "old" {
			content.OldBlocks = append(content.OldBlocks, code)
		} else {
			content.CodeBlocks = append(content.CodeBlocks, models.CodeBlock{
				Lang: lang,
				Code: code,
			})
		}
	}

	// Основной текст: убираем все блоки кода
	explanation := reCodeBlock.ReplaceAllString(raw, "")
	explanation = strings.TrimSpace(explanation)
	content.Explanation = explanation

	return content, nil
}

// extractFileNum извлекает номер из имени файла NNN.md.
func extractFileNum(path string) int {
	name := filepath.Base(path)
	if !regexp.MustCompile(`^\d{3}\.md$`).MatchString(name) {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSuffix(name, ".md"))
	if err != nil {
		return 0
	}
	return n
}

// extractTitle берёт первую непустую строку как заголовок
func extractTitle(raw, filePath string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "#")
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	// fallback: имя файла
	parts := strings.Split(filePath, "/")
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".md")
	return name
}
