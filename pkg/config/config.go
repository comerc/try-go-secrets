package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// z.ai API
	ZAIApiKey string `envconfig:"ZAI_API_KEY" required:"true"`
	ZAIApiURL string `envconfig:"ZAI_API_URL" default:"https://api.z.ai/api/coding/paas/v4"`

	// Gemini TTS
	GeminiAPIKey   string `envconfig:"GEMINI_API_KEY" required:"true"`
	GeminiTTSModel string `envconfig:"GEMINI_TTS_MODEL" default:"gemini-3.1-flash-tts-preview"`
	GeminiTTSVoice string `envconfig:"GEMINI_TTS_VOICE"`

	// Puppeteer
	PuppeteerURL string `envconfig:"PUPPETEER_URL"`

	// Paths
	RawDir    string `envconfig:"RAW_DIR" default:"./raw"`
	OutputDir string `envconfig:"OUTPUT_DIR" default:"./output"`
	StateDir  string `envconfig:"STATE_DIR" default:"./state"`

	// Video settings
	VideoWidth  int `envconfig:"VIDEO_WIDTH" default:"1080"`
	VideoHeight int `envconfig:"VIDEO_HEIGHT" default:"1920"`
	VideoFPS    int `envconfig:"VIDEO_FPS" default:"30"`

	// Language: ru (default), en, cn, hi, ja, es
	Lang string `envconfig:"LANG" default:"ru"`

	// LLM backend: "codex-cli" (default), "claude-cli", or "zai-api"
	LLMBackend string `envconfig:"LLM_BACKEND" default:"codex-cli"`
	LLMModel   string `envconfig:"LLM_MODEL"`
	LLMEffort  string `envconfig:"LLM_EFFORT"`
}

func Load() (*Config, error) {
	// .env не обязателен — можно задать через окружение
	_ = godotenv.Load()

	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
