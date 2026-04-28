package tests

import (
	"os"
	"path/filepath"
	"testing"

	"go-secrets-pipeline/pkg/services"
)

func TestParseMarkdown(t *testing.T) {
	// Создаём тестовый файл
	content := `В Go можно вернуть несколько значений из функции.

` + "```go" + `
func divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, fmt.Errorf("деление на ноль")
    }
    return a / b, nil
}
` + "```" + `

` + "```old" + `
// old way
result, _ := divide(10, 2)
` + "```"

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test__line-042.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rc, err := services.ParseMarkdown(filePath)
	if err != nil {
		t.Fatalf("ParseMarkdown error: %v", err)
	}

	if rc.FileNum != 42 {
		t.Errorf("FileNum = %d, want 42", rc.FileNum)
	}
	if len(rc.CodeBlocks) != 1 {
		t.Errorf("CodeBlocks = %d, want 1", len(rc.CodeBlocks))
	}
	if rc.CodeBlocks[0].Lang != "go" {
		t.Errorf("Lang = %q, want go", rc.CodeBlocks[0].Lang)
	}
	if len(rc.OldBlocks) != 1 {
		t.Errorf("OldBlocks = %d, want 1", len(rc.OldBlocks))
	}
	if rc.Explanation == "" {
		t.Error("Explanation пустой")
	}
}
