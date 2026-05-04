package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"go-secrets-pipeline/pkg/config"
	"go-secrets-pipeline/pkg/models"
	"go-secrets-pipeline/pkg/services"
)

type VideoGenerator struct {
	cfg   *config.Config
	tts   *services.TTSService
	video *services.VideoService
}

func NewVideoGenerator(cfg *config.Config, tts *services.TTSService, video *services.VideoService) *VideoGenerator {
	return &VideoGenerator{cfg: cfg, tts: tts, video: video}
}

// Generate создаёт финальный MP4 из сценария и контента
func (g *VideoGenerator) Generate(script *models.Script, content *models.RawContent) (*models.ProductionResult, error) {
	date := time.Now().Format("2006-01-02")
	return g.GenerateForDate(script, content, date)
}

// GenerateForDate создаёт финальный MP4 с датой артефактов YYYY-MM-DD.
func (g *VideoGenerator) GenerateForDate(script *models.Script, content *models.RawContent, date string) (*models.ProductionResult, error) {
	result := &models.ProductionResult{
		FileNum: script.FileNum,
		Slug:    script.Slug,
	}

	// 1. Синтез голоса
	audioDir := filepath.Join(g.cfg.OutputDir, "audio")
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return nil, err
	}
	audioPath := filepath.Join(audioDir, fmt.Sprintf("%s__%03d.wav", date, script.FileNum))

	voice, err := g.tts.SelectVoice(script.Voice)
	if err != nil {
		return nil, fmt.Errorf("выбор голоса TTS: %w", err)
	}
	script.Voice = voice

	fmt.Printf("  🎙  Синтез голоса (Audio-Tags, %s)...\n", voice)
	ttsText := script.NarrationTags
	if ttsText == "" {
		ttsText = script.NarrationText
	}
	if err := g.tts.Synthesize(ttsText, voice, audioPath); err != nil {
		return nil, fmt.Errorf("TTS синтез: %w", err)
	}
	result.AudioPath = audioPath

	// Измеряем реальную длительность аудио для точной синхронизации субтитров
	actualAudioDur, err := g.video.GetDuration(audioPath)
	if err != nil || actualAudioDur <= 0 {
		actualAudioDur = script.TotalSeconds // fallback на оценку
	}
	// Масштабный коэффициент: реальная скорость TTS vs оценочная
	timeScale := actualAudioDur / script.TotalSeconds
	fmt.Printf("  ✓  Аудио: %s (оценка %.1fs → реально %.1fs, scale=%.2f)\n",
		audioPath, script.TotalSeconds, actualAudioDur, timeScale)

	// 2. Рендер видео с кодом (если есть блоки кода)
	videoPath := filepath.Join(g.cfg.OutputDir, "videos", fmt.Sprintf("%s__%03d.mp4", date, script.FileNum))
	videoPath, err = filepath.Abs(videoPath)
	if err != nil {
		return nil, fmt.Errorf("абсолютный путь финального видео: %w", err)
	}
	result.VideoPath = videoPath

	// Код для отображения: приоритет у script.DisplayCode (сгенерирован LLM),
	// fallback — оригинальный код из raw-файла
	displayCode := script.Code
	displayLang := script.CodeLang
	if displayCode == "" {
		if len(content.CodeBlocks) > 0 {
			displayCode = content.CodeBlocks[0].Code
			displayLang = content.CodeBlocks[0].Lang
		} else if len(content.OldBlocks) > 0 {
			displayCode = content.OldBlocks[0]
		}
	}
	if displayLang == "" {
		displayLang = "go"
	}

	if displayCode != "" {
		spec := &models.VideoSpec{
			Slug:          script.Slug,
			AudioPath:     audioPath,
			OutputPath:    videoPath,
			Width:         g.cfg.VideoWidth,
			Height:        g.cfg.VideoHeight,
			FPS:           g.cfg.VideoFPS,
			PlaylistTitle: g.cfg.PlaylistTitle,
		}

		fmt.Printf("  🎬  Рендер видео (Puppeteer)...\n")
		rawVideoPath, err := g.video.RenderCodeVideo(spec, displayCode, displayLang, actualAudioDur, buildSubtitleWords(script.Segments, timeScale))
		if err != nil {
			return nil, fmt.Errorf("рендер видео: %w", err)
		}

		// 3. Объединение аудио + видео
		fmt.Printf("  🎞  Объединение аудио+видео (FFmpeg)...\n")
		if err := g.video.MergeAudioVideo(rawVideoPath, audioPath, videoPath); err != nil {
			return nil, fmt.Errorf("FFmpeg merge: %w", err)
		}

		// Удаляем промежуточный raw файл
		os.Remove(rawVideoPath)
	} else {
		// Нет кода — создаём видео только из аудио со статичным фоном
		fmt.Printf("  🎞  Создание видео из аудио (без кода)...\n")
		if err := g.createAudioOnlyVideo(audioPath, videoPath); err != nil {
			return nil, err
		}
	}

	// 4. Получаем длительность финального видео
	dur, err := g.video.GetDuration(videoPath)
	if err != nil {
		fmt.Printf("  ⚠  Не удалось получить длительность: %v\n", err)
	}
	result.DurationSec = dur
	result.Success = true

	fmt.Printf("  ✓  Видео: %s (%.1fs)\n", videoPath, dur)
	return result, nil
}

// buildSubtitleWords разбивает сегменты сценария на слова с временными метками.
// Внутри каждого сегмента слова распределяются пропорционально длине (в рунах).
// Символ + (ударение для TTS) убирается из отображаемого текста.
func buildSubtitleWords(segments []models.Segment, timeScale float64) []services.SubtitleWord {
	var words []services.SubtitleWord
	for _, seg := range segments {
		segWords := strings.Fields(seg.Text)
		if len(segWords) == 0 {
			continue
		}
		totalRunes := 0
		for _, w := range segWords {
			totalRunes += utf8.RuneCountInString(w)
		}
		if totalRunes == 0 {
			continue
		}
		runeAccum := 0
		for _, w := range segWords {
			t := (seg.StartSec + float64(runeAccum)/float64(totalRunes)*seg.DurationSec) * timeScale
			words = append(words, services.SubtitleWord{
				Word:     w,
				StartSec: t,
			})
			runeAccum += utf8.RuneCountInString(w)
		}
	}
	return words
}

// createAudioOnlyVideo создаёт видео с тёмным фоном и только аудио
func (g *VideoGenerator) createAudioOnlyVideo(audioPath, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	// FFmpeg: генерируем чёрный фон + аудио
	cmd := fmt.Sprintf(
		`ffmpeg -y -f lavfi -i color=c=black:s=%dx%d:r=%d -i "%s" -c:v libx264 -pix_fmt yuv420p -c:a aac -b:a 192k -shortest "%s"`,
		g.cfg.VideoWidth, g.cfg.VideoHeight, g.cfg.VideoFPS, audioPath, outputPath,
	)

	return runShellCmd(cmd)
}
