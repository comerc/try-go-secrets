package models

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ParsedContent represents the parsed content from a markdown file
type ParsedContent struct {
	FilePath     string       `json:"file_path"`
	Title        string       `json:"title"`
	Explanation  string       `json:"explanation"`
	CodeBlocks   []CodeBlock  `json:"code_blocks,omitempty"`
	Diagrams     []Diagram     `json:"diagrams,omitempty"`
	Legacy       *Legacy      `json:"legacy,omitempty"`
}

// CodeBlock represents a code block from markdown
type CodeBlock struct {
	Content  string `json:"content"`
	Language string `json:"language"`
}

// Diagram represents a diagram from markdown
type Diagram struct {
	Content string `json:"content"`
	Type    string `json:"type"` // mermaid, etc.
}

// Legacy represents old/legacy content from markdown
type Legacy struct {
	Content string `json:"content"`
}

// GetNumber extracts the line number from the filename
// Example: "Go_Secret__line-043.md" -> 43
func (pc *ParsedContent) GetNumber() int {
	_, filename := filepath.Split(pc.FilePath)
	// Remove .md extension
	name := filename[:len(filename)-3]
	// Extract number after "line-"
	var num int
	_, err := fmt.Sscanf(name, "*-line-%d.md", &num)
	if err != nil {
		return 0
	}
	return num
}

// GetSlug returns a URL-safe slug from the title
func (pc *ParsedContent) GetSlug() string {
	slug := pc.Title
	slug = strings.ToLower(slug)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.ReplaceAll(slug, "/", "-")
	// Limit slug length
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return slug
}

func init() {
	// Ensure string functions are available
	// This is a placeholder - the actual implementation will use strings package
}
