package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// z.ai API
	ZAIApiKey string
	ZAIApiURL string
	ZAIModel  string

	// SaluteSpeech
	SaluteSpeechClientID     string
	SaluteSpeechClientSecret string
	SaluteSpeechScope        string
	SaluteSpeechVoice        string

	// Puppeteer
	PuppeteerURL string

	// Paths
	RawDir    string
	OutputDir string
	StateDir  string

	// Video settings
	VideoWidth  int
	VideoHeight int
	VideoFPS    int

	// Language: ru (default), en, cn, hi, ja, es
	Lang string

	// LLM backend: "zai" (default) or "claude-cli"
	LLMBackend string
}

func Load() (*Config, error) {
	// .env не обязателен — можно задать через окружение
	_ = godotenv.Load()

	cfg := &Config{
		ZAIApiKey: os.Getenv("ZAI_API_KEY"),
		ZAIApiURL: getEnvOrDefault("ZAI_API_URL", "https://api.z.ai/api/coding/paas/v4"),
		ZAIModel:  getEnvOrDefault("ZAI_MODEL", "GLM-5.1"),

		SaluteSpeechClientID:     os.Getenv("SALUTESPEECH_CLIENT_ID"),
		SaluteSpeechClientSecret: os.Getenv("SALUTESPEECH_CLIENT_SECRET"),
		SaluteSpeechScope:        getEnvOrDefault("SALUTESPEECH_SCOPE", "SALUTE_SPEECH_PERS"),
		SaluteSpeechVoice:        getEnvOrDefault("SALUTESPEECH_VOICE", "Nec_24000"),

		PuppeteerURL: getEnvOrDefault("PUPPETEER_URL", "http://localhost:3333"),

		RawDir:    getEnvOrDefault("RAW_DIR", "./raw"),
		OutputDir: getEnvOrDefault("OUTPUT_DIR", "./output"),
		StateDir:  getEnvOrDefault("STATE_DIR", "./state"),

		VideoWidth:  getEnvInt("VIDEO_WIDTH", 1080),
		VideoHeight: getEnvInt("VIDEO_HEIGHT", 1920),
		VideoFPS:    getEnvInt("VIDEO_FPS", 30),

		Lang: getEnvOrDefault("LANG", "ru"),

		LLMBackend: getEnvOrDefault("LLM_BACKEND", "zai"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.ZAIApiKey == "" {
		return fmt.Errorf("ZAI_API_KEY не задан")
	}
	if c.SaluteSpeechClientID == "" {
		return fmt.Errorf("SALUTESPEECH_CLIENT_ID не задан")
	}
	if c.SaluteSpeechClientSecret == "" {
		return fmt.Errorf("SALUTESPEECH_CLIENT_SECRET не задан")
	}
	return nil
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
