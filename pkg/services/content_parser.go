package services

import (
	"os"
	"regexp"
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

// extractFileNum извлекает номер из имени файла (*__line-NNN.md или *-line-NNN.md)
func extractFileNum(path string) int {
	re := regexp.MustCompile(`(?:__|-)line-(\d+)`)
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	m := re.FindStringSubmatch(name)
	if m == nil {
		return 0
	}
	n := 0
	for _, ch := range m[1] {
		n = n*10 + int(ch-'0')
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
