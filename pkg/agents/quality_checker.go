package agents

import (
	"fmt"
	"os"

	"go-secrets-pipeline/pkg/models"
	"go-secrets-pipeline/pkg/services"
)

const (
	maxDurationLimit = 60.0
	minDurationLimit = 5.0
	maxFileSizeMB    = 100
)

type QualityChecker struct {
	video *services.VideoService
}

func NewQualityChecker(video *services.VideoService) *QualityChecker {
	return &QualityChecker{video: video}
}

// Check проверяет результат производства
func (q *QualityChecker) Check(result *models.ProductionResult) error {
	if !result.Success {
		return fmt.Errorf("видео не было создано")
	}

	// Проверяем существование файла
	info, err := os.Stat(result.VideoPath)
	if err != nil {
		return fmt.Errorf("файл видео не найден: %w", err)
	}

	// Размер файла
	sizeMB := float64(info.Size()) / (1024 * 1024)
	if sizeMB > maxFileSizeMB {
		return fmt.Errorf("размер файла %.1f MB превышает лимит %d MB", sizeMB, maxFileSizeMB)
	}

	// Длительность
	dur, err := q.video.GetDuration(result.VideoPath)
	if err != nil {
		return fmt.Errorf("не удалось получить длительность: %w", err)
	}

	if dur >= maxDurationLimit {
		return fmt.Errorf("длительность %.1fs >= 60s (лимит YouTube Shorts)", dur)
	}
	if dur < minDurationLimit {
		return fmt.Errorf("длительность %.1fs слишком мала", dur)
	}

	result.DurationSec = dur
	fmt.Printf("  ✓  Качество OK: %.1fs, %.1f MB\n", dur, sizeMB)
	return nil
}
