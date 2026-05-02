package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go-secrets-pipeline/pkg/agents"
	"go-secrets-pipeline/pkg/config"
	"go-secrets-pipeline/pkg/models"
	"go-secrets-pipeline/pkg/services"
	"go-secrets-pipeline/pkg/state"
)

type Orchestrator struct {
	cfg       *config.Config
	processed *state.ProcessedState
	ttsUsage  *state.TTSUsageState
	llm       *services.LLMService
	tts       *services.TTSService
	video     *services.VideoService

	writer  *agents.ScriptWriter
	checker *agents.QualityChecker
	gen     *agents.VideoGenerator
}

func New(cfg *config.Config) (*Orchestrator, error) {
	ps, err := state.LoadProcessed(cfg.StateDir)
	if err != nil {
		return nil, fmt.Errorf("загрузка processed state: %w", err)
	}

	ttsUsage, err := state.LoadTTSUsage(cfg.StateDir)
	if err != nil {
		return nil, fmt.Errorf("загрузка tts_usage state: %w", err)
	}

	llm := services.NewLLMService(cfg)
	tts := services.NewTTSService(cfg)
	video := services.NewVideoService(cfg)

	return &Orchestrator{
		cfg:       cfg,
		processed: ps,
		ttsUsage:  ttsUsage,
		llm:       llm,
		tts:       tts,
		video:     video,
		writer:    agents.NewScriptWriter(llm, tts, cfg.Lang),
		checker:   agents.NewQualityChecker(video),
		gen:       agents.NewVideoGenerator(cfg, tts, video),
	}, nil
}

// Run запускает полный пайплайн для одного файла
func (o *Orchestrator) Run(fileNum int) (*models.ProductionResult, error) {
	log.SetFlags(log.Ltime)
	start := time.Now()

	fmt.Printf("\n━━━ Go Secrets Pipeline ━━━\n")
	if fileNum > 0 {
		fmt.Printf("Файл: #%d\n\n", fileNum)
	} else {
		fmt.Printf("Файл: случайный\n\n")
	}

	// 1. Выбор контента
	fmt.Printf("[1/5] Выбор контента...\n")
	filePath, num, err := agents.SelectContent(o.cfg.RawDir, o.processed, fileNum)
	if err != nil {
		return nil, fmt.Errorf("выбор контента: %w", err)
	}
	fmt.Printf("  ✓  Выбран файл: %s (#%d)\n", filepath.Base(filePath), num)

	// 2. Парсинг контента
	fmt.Printf("[2/5] Парсинг контента...\n")
	content, err := services.ParseMarkdown(filePath)
	if err != nil {
		return nil, fmt.Errorf("парсинг markdown: %w", err)
	}
	fmt.Printf("  ✓  Блоков кода: %d, OldBlocks: %d\n", len(content.CodeBlocks), len(content.OldBlocks))

	// 3. Генерация сценария
	fmt.Printf("[3/5] Генерация сценария (%s)...\n", o.cfg.LLMBackend)
	script, err := o.writer.Write(content)
	if err != nil {
		return nil, fmt.Errorf("генерация сценария: %w", err)
	}
	fmt.Printf("  ✓  Slug: %s, длина нарратива: %d симв (~%.0fs)\n",
		script.Slug, len([]rune(script.NarrationText)), script.TotalSeconds)

	// Проверяем TTS лимит (информационно, не блокируем)
	todayChars := o.ttsUsage.TodayUsage()
	narrationChars := len([]rune(script.NarrationText))
	fmt.Printf("  ℹ  TTS сегодня: %d символов использовано\n", todayChars)

	// Сохраняем сценарий
	scriptPath, err := o.writer.Save(script, o.cfg.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("сохранение сценария: %w", err)
	}
	fmt.Printf("  ✓  Сценарий: %s\n", scriptPath)

	// 4. Генерация видео
	fmt.Printf("[4/5] Генерация видео...\n")
	result, err := o.gen.Generate(script, content)
	if err != nil {
		return nil, fmt.Errorf("генерация видео: %w", err)
	}
	result.ScriptPath = scriptPath

	// 5. Контроль качества
	fmt.Printf("[5/5] Контроль качества...\n")
	if err := o.checker.Check(result); err != nil {
		return result, fmt.Errorf("контроль качества: %w", err)
	}

	// Обновляем состояние
	if err := o.processed.Add(num); err != nil {
		log.Printf("⚠ Ошибка записи processed.json: %v", err)
	}
	if err := o.ttsUsage.Add(narrationChars); err != nil {
		log.Printf("⚠ Ошибка записи tts_usage.json: %v", err)
	}

	elapsed := time.Since(start).Round(time.Second)
	fmt.Printf("\n✅ Готово за %s!\n", elapsed)
	fmt.Printf("   Видео: %s\n", result.VideoPath)
	fmt.Printf("   Длительность: %.1fs\n\n", result.DurationSec)

	return result, nil
}

// ensureOutputDirs создаёт необходимые директории
func ensureOutputDirs(cfg *config.Config) error {
	dirs := []string{
		filepath.Join(cfg.OutputDir, "scripts"),
		filepath.Join(cfg.OutputDir, "audio"),
		filepath.Join(cfg.OutputDir, "videos"),
		filepath.Join(cfg.OutputDir, "logs"),
		cfg.StateDir,
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

// Init создаёт все нужные директории
func Init(cfg *config.Config) error {
	return ensureOutputDirs(cfg)
}

// RunFix перегенерирует аудио и видео из существующего сценария (без запроса к LLM).
// Используется после ручной правки output/scripts/*__NNN.json.
func (o *Orchestrator) RunFix(fileNum int) (*models.ProductionResult, error) {
	log.SetFlags(log.Ltime)
	fmt.Printf("\n━━━ Fix: перегенерация #%d ━━━\n\n", fileNum)

	// Находим файл сценария по номеру
	pattern := filepath.Join(o.cfg.OutputDir, "scripts", fmt.Sprintf("*__%03d.json", fileNum))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("сценарий #%d не найден в %s", fileNum, filepath.Dir(pattern))
	}
	scriptPath := matches[len(matches)-1] // берём последний если несколько
	fmt.Printf("  ✓  Сценарий: %s\n", scriptPath)

	// Загружаем сценарий из JSON
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("чтение сценария: %w", err)
	}
	var script models.Script
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, fmt.Errorf("парсинг сценария: %w", err)
	}

	// Генерируем аудио + видео (без LLM)
	fmt.Printf("[1/2] Генерация видео...\n")
	result, err := o.gen.Generate(&script, &models.RawContent{FileNum: script.FileNum})
	if err != nil {
		return nil, fmt.Errorf("генерация видео: %w", err)
	}
	result.ScriptPath = scriptPath

	// Контроль качества
	fmt.Printf("[2/2] Контроль качества...\n")
	if err := o.checker.Check(result); err != nil {
		return result, fmt.Errorf("контроль качества: %w", err)
	}

	fmt.Printf("\n✅ Готово: %s (%.1fs)\n\n", result.VideoPath, result.DurationSec)
	return result, nil
}
