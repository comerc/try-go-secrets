package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-secrets-pipeline/pkg/agents"
	"go-secrets-pipeline/pkg/config"
	"go-secrets-pipeline/pkg/models"
	"go-secrets-pipeline/pkg/services"
	"go-secrets-pipeline/pkg/state"
)

type Orchestrator struct {
	cfg        *config.Config
	processed  *state.ProcessedState
	ttsUsage   *state.TTSUsageState
	ytSchedule *state.YouTubeScheduleState
	llm        *services.LLMService
	tts        *services.TTSService
	video      *services.VideoService
	youtube    *services.YouTubeService

	writer  *agents.ScriptWriter
	checker *agents.QualityChecker
	gen     *agents.VideoGenerator
}

func New(cfg *config.Config) (*Orchestrator, error) {
	langStateDir := cfg.LangStateDir()

	ps, err := state.LoadProcessed(langStateDir)
	if err != nil {
		return nil, fmt.Errorf("загрузка processed state: %w", err)
	}

	ttsUsage, err := state.LoadTTSUsage(cfg.StateDir)
	if err != nil {
		return nil, fmt.Errorf("загрузка tts_usage state: %w", err)
	}

	ytSchedule, err := state.LoadYouTubeSchedule(langStateDir)
	if err != nil {
		return nil, fmt.Errorf("загрузка youtube_schedule state: %w", err)
	}

	llm := services.NewLLMService(cfg)
	tts := services.NewTTSService(cfg)
	video := services.NewVideoService(cfg)
	youtubeSvc := services.NewYouTubeService(cfg)

	orch := &Orchestrator{
		cfg:        cfg,
		processed:  ps,
		ttsUsage:   ttsUsage,
		ytSchedule: ytSchedule,
		llm:        llm,
		tts:        tts,
		video:      video,
		youtube:    youtubeSvc,
		writer:     agents.NewScriptWriter(llm, tts, cfg.VideoLang),
		checker:    agents.NewQualityChecker(video),
		gen:        agents.NewVideoGenerator(cfg, tts, video),
	}
	orch.gen.SetTTSUsageRecorder(orch.recordTTSUsage)
	return orch, nil
}

func (o *Orchestrator) recordTTSUsage(chars int) error {
	before := o.ttsUsage.TodayUsage()
	after, err := o.ttsUsage.Add(chars)
	if err != nil {
		return err
	}
	fmt.Printf("  ℹ  TTS сегодня: %d + 1 = %d запусков, %d + %d = %d символов\n",
		before.Runs, after.Runs, before.Chars, chars, after.Chars)
	return nil
}

// Run запускает полный пайплайн для одного файла
func (o *Orchestrator) Run(fileNum int) (*models.ProductionResult, error) {
	log.SetFlags(log.Ltime)
	start := time.Now()

	fmt.Printf("\n━━━ Go Secrets Pipeline ━━━\n")
	fmt.Printf("Язык: %s\n", o.cfg.VideoLang)
	if fileNum > 0 {
		fmt.Printf("Файл: #%d\n\n", fileNum)
	} else {
		fmt.Printf("Файл: случайный\n\n")
	}

	// 1. Выбор контента
	steps := 5
	if o.cfg.YouTubeEnabled {
		steps = 6
	}

	fmt.Printf("[1/%d] Выбор контента...\n", steps)
	filePath, num, err := agents.SelectContent(o.cfg.RawDir, o.processed, fileNum)
	if err != nil {
		return nil, fmt.Errorf("выбор контента: %w", err)
	}
	fmt.Printf("  ✓  Выбран файл: %s (#%d)\n", filepath.Base(filePath), num)

	// 2. Парсинг контента
	fmt.Printf("[2/%d] Парсинг контента...\n", steps)
	content, err := services.ParseMarkdown(filePath)
	if err != nil {
		return nil, fmt.Errorf("парсинг markdown: %w", err)
	}
	fmt.Printf("  ✓  Блоков кода: %d, OldBlocks: %d\n", len(content.CodeBlocks), len(content.OldBlocks))

	// 3. Генерация сценария
	fmt.Printf("[3/%d] Генерация сценария (%s)...\n", steps, o.cfg.LLMBackend)
	script, err := o.writer.Write(content)
	if err != nil {
		return nil, fmt.Errorf("генерация сценария: %w", err)
	}
	fmt.Printf("  ✓  Длина нарратива: %d симв (~%.0fs)\n",
		len([]rune(script.NarrationText)), script.TotalSeconds)

	// Сохраняем сценарий
	scriptPath, err := o.writer.Save(script, o.cfg.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("сохранение сценария: %w", err)
	}
	fmt.Printf("  ✓  Сценарий: %s\n", scriptPath)

	// 4. Генерация видео
	fmt.Printf("[4/%d] Генерация видео...\n", steps)
	result, err := o.gen.Generate(script, content)
	if err != nil {
		return nil, fmt.Errorf("генерация видео: %w", err)
	}
	result.ScriptPath = scriptPath

	// 5. Контроль качества
	fmt.Printf("[5/%d] Контроль качества...\n", steps)
	if err := o.checker.Check(result); err != nil {
		return result, fmt.Errorf("контроль качества: %w", err)
	}

	// Фиксируем обработку после успешного создания и QC, независимо от YouTube.
	if err := o.processed.Add(num); err != nil {
		log.Printf("✗ Ошибка записи processed.json: %v", err)
	}

	if o.cfg.YouTubeEnabled {
		fmt.Printf("[6/%d] Публикация YouTube...\n", steps)
		if err := o.publishToYouTube(script, result); err != nil {
			return result, fmt.Errorf("публикация YouTube: %w", err)
		}
	}

	elapsed := time.Since(start).Round(time.Second)
	fmt.Printf("\n✅ Готово за %s!\n", elapsed)
	fmt.Printf("   Видео: %s\n", result.VideoPath)
	fmt.Printf("   Длительность: %.1fs\n\n", result.DurationSec)

	return result, nil
}

func (o *Orchestrator) publishToYouTube(script *models.Script, result *models.ProductionResult) error {
	publishAt, err := o.nextPublishTime()
	if err != nil {
		return err
	}
	date := publishAt.Format("2006-01-02")
	id := state.FormatVideoID(script.FileNum)

	upload, err := o.youtube.Upload(context.Background(), services.YouTubeUploadRequest{
		Script:    script,
		Result:    result,
		PublishAt: publishAt,
	})
	if err != nil {
		return err
	}
	if err := o.ytSchedule.Reserve(id, date); err != nil {
		return fmt.Errorf("запись даты публикации: %w", err)
	}

	fmt.Printf("  ✓  YouTube: %s\n", upload.VideoURL)
	fmt.Printf("  ✓  Имя файла: %s\n", upload.FileName)
	fmt.Printf("  ✓  Запланировано: %s\n", upload.PublishAt.Format(time.RFC3339))
	if upload.PlaylistWarning != "" {
		fmt.Printf("  ✗  Ролик загружен, но не добавлен в плейлист: %s\n", upload.PlaylistWarning)
	}
	return nil
}

func (o *Orchestrator) nextPublishTime() (time.Time, error) {
	loc := time.Local
	if o.cfg.YouTubeScheduleLocation != "" {
		loaded, err := time.LoadLocation(o.cfg.YouTubeScheduleLocation)
		if err != nil {
			return time.Time{}, fmt.Errorf("таймзона YouTube: %w", err)
		}
		loc = loaded
	}
	hour, minute, err := parseScheduleTime(o.cfg.YouTubeScheduleTime)
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now().In(loc)
	candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	if !candidate.After(now) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	for o.ytSchedule.IsReserved(candidate.Format("2006-01-02")) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate, nil
}

func parseScheduleTime(value string) (int, int, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("YOUTUBE_SCHEDULE_TIME должен быть в формате HH:MM")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("YOUTUBE_SCHEDULE_TIME: час %q: %w", parts[0], err)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("YOUTUBE_SCHEDULE_TIME: минута %q: %w", parts[1], err)
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("YOUTUBE_SCHEDULE_TIME должен быть в диапазоне 00:00..23:59")
	}
	return hour, minute, nil
}

// PublishExisting публикует уже готовый ролик из output/<lang>/scripts и output/<lang>/videos.
func (o *Orchestrator) PublishExisting(fileNum int) (*services.YouTubeUploadResult, error) {
	log.SetFlags(log.Ltime)
	fmt.Printf("\n━━━ YouTube публикация #%d ━━━\n\n", fileNum)
	fmt.Printf("Язык: %s\n\n", o.cfg.VideoLang)

	script, scriptPath, err := o.loadLatestScript(fileNum)
	if err != nil {
		return nil, err
	}
	fmt.Printf("  ✓  Сценарий: %s\n", scriptPath)

	videoPath, err := o.findLatestVideo(fileNum)
	if err != nil {
		return nil, err
	}
	fmt.Printf("  ✓  Видео: %s\n", videoPath)

	result := &models.ProductionResult{
		FileNum:    script.FileNum,
		VideoPath:  videoPath,
		ScriptPath: scriptPath,
		Success:    true,
	}

	publishAt, err := o.nextPublishTime()
	if err != nil {
		return nil, err
	}
	date := publishAt.Format("2006-01-02")
	id := state.FormatVideoID(script.FileNum)
	if o.ytSchedule.IsPublished(id) {
		return nil, fmt.Errorf("ролик #%s уже есть в youtube_schedule.json", id)
	}

	upload, err := o.youtube.Upload(context.Background(), services.YouTubeUploadRequest{
		Script:    script,
		Result:    result,
		PublishAt: publishAt,
	})
	if err != nil {
		return nil, err
	}
	if err := o.ytSchedule.Reserve(id, date); err != nil {
		return nil, fmt.Errorf("запись даты публикации: %w", err)
	}

	fmt.Printf("  ✓  YouTube: %s\n", upload.VideoURL)
	fmt.Printf("  ✓  Имя файла: %s\n", upload.FileName)
	fmt.Printf("  ✓  Запланировано: %s\n\n", upload.PublishAt.Format(time.RFC3339))
	if upload.PlaylistWarning != "" {
		fmt.Printf("  ✗  Ролик загружен, но не добавлен в плейлист: %s\n", upload.PlaylistWarning)
	}
	return upload, nil
}

func (o *Orchestrator) loadLatestScript(fileNum int) (*models.Script, string, error) {
	pattern := filepath.Join(o.cfg.OutputDir, "scripts", fmt.Sprintf("*__%03d.json", fileNum))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil, "", fmt.Errorf("сценарий #%d не найден в %s", fileNum, filepath.Dir(pattern))
	}
	scriptPath := matches[len(matches)-1]

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, "", fmt.Errorf("чтение сценария: %w", err)
	}
	var script models.Script
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, "", fmt.Errorf("парсинг сценария: %w", err)
	}
	return &script, scriptPath, nil
}

func (o *Orchestrator) findLatestVideo(fileNum int) (string, error) {
	pattern := filepath.Join(o.cfg.OutputDir, "videos", fmt.Sprintf("*__%03d.mp4", fileNum))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("видео #%d не найдено в %s", fileNum, filepath.Dir(pattern))
	}
	videoPath := matches[len(matches)-1]
	abs, err := filepath.Abs(videoPath)
	if err != nil {
		return "", fmt.Errorf("абсолютный путь видео: %w", err)
	}
	return abs, nil
}

// ensureOutputDirs создаёт необходимые директории
func ensureOutputDirs(cfg *config.Config) error {
	dirs := []string{
		filepath.Join(cfg.OutputDir, "scripts"),
		filepath.Join(cfg.OutputDir, "audio"),
		filepath.Join(cfg.OutputDir, "videos"),
		filepath.Join(cfg.OutputDir, "raw"),
		cfg.StateDir,
		cfg.LangStateDir(),
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
// Используется после ручной правки output/<lang>/scripts/*__NNN.json.
func (o *Orchestrator) RunFix(fileNum int) (*models.ProductionResult, error) {
	log.SetFlags(log.Ltime)
	fmt.Printf("\n━━━ Fix: перегенерация #%d ━━━\n\n", fileNum)
	fmt.Printf("Язык: %s\n\n", o.cfg.VideoLang)

	// Находим файл сценария по номеру
	pattern := filepath.Join(o.cfg.OutputDir, "scripts", fmt.Sprintf("*__%03d.json", fileNum))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("сценарий #%d не найден в %s", fileNum, filepath.Dir(pattern))
	}
	scriptPath := matches[len(matches)-1] // берём последний если несколько
	fmt.Printf("  ✓  Сценарий: %s\n", scriptPath)
	artifactDate := artifactDateFromPath(scriptPath)
	fmt.Printf("  ✓  Дата артефактов: %s\n", artifactDate)

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
	result, err := o.gen.GenerateForDate(&script, &models.RawContent{FileNum: script.FileNum}, artifactDate)
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

func artifactDateFromPath(path string) string {
	base := filepath.Base(path)
	if len(base) >= len("2006-01-02") {
		return base[:len("2006-01-02")]
	}
	return time.Now().Format("2006-01-02")
}
