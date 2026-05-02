package agents

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"go-secrets-pipeline/pkg/state"
)

var reLineNum = regexp.MustCompile(`^(\d{3})\.md$`)

// SelectContent выбирает markdown-файл по номеру или случайно из необработанных
func SelectContent(rawDir string, ps *state.ProcessedState, num int) (string, int, error) {
	files, err := filepath.Glob(filepath.Join(rawDir, "*.md"))
	if err != nil {
		return "", 0, fmt.Errorf("ошибка чтения директории raw: %w", err)
	}
	if len(files) == 0 {
		return "", 0, fmt.Errorf("в директории %s нет .md файлов", rawDir)
	}

	if num > 0 {
		return selectByNum(files, num, ps)
	}
	return selectRandom(files, ps)
}

func selectByNum(files []string, num int, ps *state.ProcessedState) (string, int, error) {
	target := fmt.Sprintf("%03d", num)
	for _, f := range files {
		fileNum, ok := parseFileNum(f)
		if !ok {
			continue
		}
		if fmt.Sprintf("%03d", fileNum) == target {
			if ps.IsProcessed(num) {
				return "", 0, fmt.Errorf("файл #%d уже обработан", num)
			}
			return f, num, nil
		}
	}
	return "", 0, fmt.Errorf("файл с номером %d не найден", num)
}

func selectRandom(files []string, ps *state.ProcessedState) (string, int, error) {
	var unprocessed []struct {
		path string
		num  int
	}

	for _, f := range files {
		n, ok := parseFileNum(f)
		if !ok {
			continue
		}
		if !ps.IsProcessed(n) {
			unprocessed = append(unprocessed, struct {
				path string
				num  int
			}{f, n})
		}
	}

	if len(unprocessed) == 0 {
		return "", 0, fmt.Errorf("все файлы уже обработаны")
	}

	// Случайный выбор с воспроизводимостью через os entropy
	idx := rand.Intn(len(unprocessed))
	chosen := unprocessed[idx]
	return chosen.path, chosen.num, nil
}

func parseFileNum(path string) (int, bool) {
	base := filepath.Base(path)
	m := reLineNum.FindStringSubmatch(base)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

// FileExists проверяет существование файла
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
