package tests

import (
	"os"
	"path/filepath"
	"testing"

	"go-secrets-pipeline/pkg/agents"
	"go-secrets-pipeline/pkg/state"
)

func TestSelectByNum(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаём тестовые файлы
	files := []string{
		"003.md",
		"043.md",
		"100.md",
	}
	for _, f := range files {
		os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644)
	}

	stateDir := t.TempDir()
	ps, _ := state.LoadProcessed(stateDir)

	// Выбор по номеру
	path, num, err := agents.SelectContent(tmpDir, ps, 43)
	if err != nil {
		t.Fatalf("SelectContent error: %v", err)
	}
	if num != 43 {
		t.Errorf("num = %d, want 43", num)
	}
	if filepath.Base(path) != "043.md" {
		t.Errorf("path = %s", filepath.Base(path))
	}

	// Файл уже обработан
	if err := ps.Add(43); err != nil {
		t.Fatalf("processed add: %v", err)
	}
	_, _, err = agents.SelectContent(tmpDir, ps, 43)
	if err == nil {
		t.Error("ожидалась ошибка для уже обработанного файла")
	}
}

func TestSelectRandom(t *testing.T) {
	tmpDir := t.TempDir()
	for _, f := range []string{
		"001.md",
		"002.md",
	} {
		os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644)
	}

	stateDir := t.TempDir()
	ps, _ := state.LoadProcessed(stateDir)

	_, num, err := agents.SelectContent(tmpDir, ps, 0)
	if err != nil {
		t.Fatalf("random select error: %v", err)
	}
	if num != 1 && num != 2 {
		t.Errorf("неожиданный номер: %d", num)
	}
}
