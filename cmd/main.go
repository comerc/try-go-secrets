package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"go-secrets-pipeline/pkg/config"
	"go-secrets-pipeline/pkg/models"
	"go-secrets-pipeline/pkg/orchestrator"
	"go-secrets-pipeline/pkg/services"
)

func main() {
	var (
		fileNum = flag.Int("num", 0, "номер файла для обработки (0 = случайный)")
		fixNum  = flag.Int("fix", 0, "перегенерировать аудио+видео из существующего сценария")
		testTTS = flag.String("test-tts", "", "тестовый текст для синтеза TTS")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Конфигурация: %v\n\nСкопируйте .env.example в .env и заполните ключи API.", err)
	}

	// Режим тестирования TTS
	if *testTTS != "" {
		tts := services.NewTTSService(cfg)
		outPath := "output/audio/test-tts.wav"
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
	if *fixNum > 0 {
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
