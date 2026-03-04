package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aka/semarang/pkg/models"
)

// ContentParser handles parsing markdown files containing Go secrets
type ContentParser struct {
	goCodeRegex     *regexp.Regexp
	mermaidRegex    *regexp.Regexp
	oldContentRegex *regexp.Regexp
	exampleRegex    *regexp.Regexp
	diagramRegex    *regexp.Regexp
}

// NewContentParser creates a new ContentParser
func NewContentParser() (*ContentParser, error) {
	goCodeRegex, err := regexp.Compile("(?s)```go\\n(.*?)\\n```")
	if err != nil {
		return nil, fmt.Errorf("failed to compile go code regex: %w", err)
	}

	mermaidRegex, err := regexp.Compile("(?s)```mermaid\\n(.*?)\\n```")
	if err != nil {
		return nil, fmt.Errorf("failed to compile mermaid regex: %w", err)
	}

	oldContentRegex, err := regexp.Compile("(?s)```old\\n(.*?)\\n```")
	if err != nil {
		return nil, fmt.Errorf("failed to compile old content regex: %w", err)
	}

	return &ContentParser{
		goCodeRegex:     goCodeRegex,
		mermaidRegex:    mermaidRegex,
		oldContentRegex: oldContentRegex,
		exampleRegex:    regexp.MustCompile(`(?i)пример:`),
		diagramRegex:    regexp.MustCompile(`(?i)диаграмма:`),
	}, nil
}

// Parse parses a markdown file and returns ParsedContent
func (cp *ContentParser) Parse(filePath string) (*models.ParsedContent, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	text := string(content)

	// Extract title from filename
	_, filename := filepath.Split(filePath)
	title := strings.TrimSuffix(filename, ".md")

	// Extract code blocks
	codeBlocks := cp.extractCodeBlocks(text)

	// Extract diagrams
	diagrams := cp.extractDiagrams(text)

	// Extract legacy content
	legacy := cp.extractLegacy(text)

	// Extract explanation (text outside code blocks, diagrams, and legacy)
	explanation := cp.extractExplanation(text)

	return &models.ParsedContent{
		FilePath:    filePath,
		Title:       title,
		Explanation: explanation,
		CodeBlocks:  codeBlocks,
		Diagrams:    diagrams,
		Legacy:      legacy,
	}, nil
}

// extractCodeBlocks extracts all Go code blocks from the text
func (cp *ContentParser) extractCodeBlocks(text string) []models.CodeBlock {
	matches := cp.goCodeRegex.FindAllStringSubmatch(text, -1)
	blocks := make([]models.CodeBlock, 0, len(matches))

	for _, match := range matches {
		if len(match) > 1 {
			blocks = append(blocks, models.CodeBlock{
				Content:  strings.TrimSpace(match[1]),
				Language: "go",
			})
		}
	}

	return blocks
}

// extractDiagrams extracts all mermaid diagrams from the text
func (cp *ContentParser) extractDiagrams(text string) []models.Diagram {
	matches := cp.mermaidRegex.FindAllStringSubmatch(text, -1)
	diagrams := make([]models.Diagram, 0, len(matches))

	for _, match := range matches {
		if len(match) > 1 {
			diagrams = append(diagrams, models.Diagram{
				Content: strings.TrimSpace(match[1]),
				Type:    "mermaid",
			})
		}
	}

	return diagrams
}

// extractLegacy extracts the legacy content from the text
func (cp *ContentParser) extractLegacy(text string) *models.Legacy {
	match := cp.oldContentRegex.FindStringSubmatch(text)
	if match != nil && len(match) > 1 {
		return &models.Legacy{
			Content: strings.TrimSpace(match[1]),
		}
	}
	return nil
}

// extractExplanation extracts the main explanation text, excluding code blocks, diagrams, and legacy content
func (cp *ContentParser) extractExplanation(text string) string {
	// Remove code blocks
	text = cp.goCodeRegex.ReplaceAllString(text, "")

	// Remove mermaid diagrams
	text = cp.mermaidRegex.ReplaceAllString(text, "")

	// Remove legacy content
	text = cp.oldContentRegex.ReplaceAllString(text, "")

	// Remove labels
	text = cp.exampleRegex.ReplaceAllString(text, "")
	text = cp.diagramRegex.ReplaceAllString(text, "")

	// Clean up whitespace
	lines := strings.Split(text, "\n")
	cleanLines := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// ParseNumberFromFilename extracts the line number from a filename
// Example: "Go_Secret__line-043.md" -> 43
func ParseNumberFromFilename(filename string) int {
	re := regexp.MustCompile(`line-(\d+)\.md$`)
	match := re.FindStringSubmatch(filename)
	if match != nil && len(match) > 1 {
		var num int
		_, err := fmt.Sscanf(match[1], "%d", &num)
		if err == nil {
			return num
		}
	}
	return 0
}
