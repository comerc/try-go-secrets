package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-secrets-pipeline/pkg/config"
	"go-secrets-pipeline/pkg/models"
	"go-secrets-pipeline/pkg/orchestrator"
	"go-secrets-pipeline/pkg/services"
)

func main() {
	var (
		fileNum          = flag.Int("num", 0, "номер файла для обработки (0 = случайный)")
		fixNum           = flag.Int("fix", 0, "перегенерировать аудио+видео из существующего сценария")
		pubNum           = flag.Int("pub", 0, "опубликовать готовое видео на YouTube")
		youtubeAuth      = flag.Bool("youtube-auth", false, "получить YouTube OAuth refresh token")
		youtubeAuthWrite = flag.Bool("youtube-auth-write", false, "получить YouTube OAuth refresh token и записать его в .env")
		testTTS          = flag.String("test-tts", "", "тестовый текст для синтеза TTS")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Конфигурация: %v\n\nСкопируйте .env.example в .env и заполните ключи API.", err)
	}

	// Режим тестирования TTS
	if *testTTS != "" {
		tts := services.NewTTSService(cfg)
		outPath := filepath.Join(cfg.OutputDir, "audio", "test-tts.wav")
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			log.Fatalf("Директория TTS: %v", err)
		}
		fmt.Printf("Синтез: %q\n", *testTTS)
		voice, err := tts.SelectVoice("")
		if err != nil {
			log.Fatalf("TTS голос: %v", err)
		}
		fmt.Printf("Голос: %s\n", voice)
		if err := tts.Synthesize(*testTTS, voice, outPath); err != nil {
			log.Fatalf("TTS ошибка: %v", err)
		}
		fmt.Printf("Аудио сохранено: %s\n", outPath)
		os.Exit(0)
	}

	if *youtubeAuth || *youtubeAuthWrite {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		yt := services.NewYouTubeService(cfg)
		refreshToken, channels, err := yt.Authorize(ctx)
		if err != nil {
			log.Fatalf("YouTube auth: %v", err)
		}
		fmt.Printf("\nYOUTUBE_REFRESH_TOKEN=%s\n", refreshToken)
		if *youtubeAuthWrite {
			if err := upsertEnvValue(".env", "YOUTUBE_REFRESH_TOKEN", refreshToken); err != nil {
				log.Fatalf("Запись .env: %v", err)
			}
			fmt.Printf("\n✓ YOUTUBE_REFRESH_TOKEN записан в .env\n")
		}
		if len(channels) == 0 {
			fmt.Printf("\n✗ YouTube API не видит каналов для этого токена.\n")
		} else {
			fmt.Printf("\nКаналы, видимые токену:\n")
			for _, ch := range channels {
				fmt.Printf("- %s %s (%s)\n", ch.Title, ch.CustomURL, ch.ID)
			}
		}
		os.Exit(0)
	}

	// Инициализация директорий
	if err := orchestrator.Init(cfg); err != nil {
		log.Fatalf("Инициализация: %v", err)
	}

	// Создание и запуск оркестратора
	orch, err := orchestrator.New(cfg)
	if err != nil {
		log.Fatalf("Создание оркестратора: %v", err)
	}

	var result *models.ProductionResult
	if *pubNum > 0 {
		upload, err := orch.PublishExisting(*pubNum)
		if err != nil {
			log.Fatalf("Ошибка публикации: %v", err)
		}
		fmt.Printf("Видео опубликовано: %s\n", upload.VideoURL)
		return
	} else if *fixNum > 0 {
		result, err = orch.RunFix(*fixNum)
	} else {
		result, err = orch.Run(*fileNum)
	}
	if err != nil {
		log.Fatalf("Ошибка пайплайна: %v", err)
	}

	if result != nil {
		fmt.Printf("Видео создано: %s\n", result.VideoPath)
	}
}

func upsertEnvValue(path, key, value string) error {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	lines := strings.Split(string(data), "\n")
	replacement := key + "=" + value
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") || strings.HasPrefix(trimmed, "export "+key+"=") {
			lines[i] = replacement
			found = true
		}
	}
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, replacement)
	}
	out := strings.Join(lines, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return os.WriteFile(path, []byte(out), 0o600)
}
