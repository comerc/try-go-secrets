package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// z.ai API
	ZAIApiKey string `envconfig:"ZAI_API_KEY"`
	ZAIApiURL string `envconfig:"ZAI_API_URL" default:"https://api.z.ai/api/coding/paas/v4"`

	// Gemini TTS
	GeminiAPIKey   string `envconfig:"GEMINI_API_KEY" required:"true"`
	GeminiTTSModel string `envconfig:"GEMINI_TTS_MODEL" default:"gemini-2.5-flash-preview-tts"`
	GeminiTTSVoice string `envconfig:"GEMINI_TTS_VOICE"`

	// Puppeteer
	PuppeteerURL string `envconfig:"PUPPETEER_URL"`

	// Paths
	RawDir    string `envconfig:"RAW_DIR" default:"./raw"`
	OutputDir string `envconfig:"OUTPUT_DIR" default:"./output"`
	StateDir  string `envconfig:"STATE_DIR" default:"./state"`

	// Video settings
	VideoWidth    int `envconfig:"VIDEO_WIDTH" default:"1080"`
	VideoHeight   int `envconfig:"VIDEO_HEIGHT" default:"1920"`
	VideoFPS      int `envconfig:"VIDEO_FPS" default:"30"`
	TerminalTitle string

	// Language: ru (default), en, cn, hi, ja, es
	VideoLang string `envconfig:"VIDEO_LANG" default:"ru"`

	// LLM backend: "codex-cli" (default), "claude-cli", or "zai-api"
	LLMBackend string `envconfig:"LLM_BACKEND" default:"codex-cli"`
	LLMModel   string `envconfig:"LLM_MODEL"`
	LLMEffort  string `envconfig:"LLM_EFFORT"`

	// YouTube publishing
	YouTubeEnabled          bool   `envconfig:"YOUTUBE_ENABLED" default:"false"`
	YouTubeClientID         string `envconfig:"YOUTUBE_CLIENT_ID"`
	YouTubeClientSecret     string `envconfig:"YOUTUBE_CLIENT_SECRET"`
	YouTubeRefreshToken     string `envconfig:"YOUTUBE_REFRESH_TOKEN"`
	YouTubePlaylistID       string
	YouTubeScheduleLocation string
	YouTubeScheduleTime     string
}

func Load() (*Config, error) {
	// .env не обязателен — можно задать через окружение
	_ = godotenv.Load()

	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}
	if err := applyLanguageConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyLanguageConfig(cfg *Config) error {
	cfg.VideoLang = strings.TrimSpace(cfg.VideoLang)
	if cfg.VideoLang == "" {
		cfg.VideoLang = "ru"
	}
	if strings.ContainsAny(cfg.VideoLang, `/\`) || cfg.VideoLang == "." || cfg.VideoLang == ".." {
		return fmt.Errorf("VIDEO_LANG %q нельзя использовать как имя директории", cfg.VideoLang)
	}

	cfg.OutputDir = appendLangDir(cfg.OutputDir, cfg.VideoLang)

	suffix := "_" + strings.ToUpper(strings.ReplaceAll(cfg.VideoLang, "-", "_"))
	cfg.TerminalTitle = envWithLang("TERMINAL_TITLE", suffix)
	cfg.YouTubePlaylistID = envWithLang("YOUTUBE_PLAYLIST_ID", suffix)
	cfg.YouTubeScheduleLocation = envWithLang("YOUTUBE_SCHEDULE_LOCATION", suffix)
	cfg.YouTubeScheduleTime = envWithLang("YOUTUBE_SCHEDULE_TIME", suffix)

	if cfg.YouTubeEnabled {
		if cfg.YouTubePlaylistID == "" {
			return fmt.Errorf("YOUTUBE_PLAYLIST_ID%s обязателен при YOUTUBE_ENABLED=true", suffix)
		}
		if cfg.YouTubeScheduleLocation == "" {
			return fmt.Errorf("YOUTUBE_SCHEDULE_LOCATION%s обязателен при YOUTUBE_ENABLED=true", suffix)
		}
		if cfg.YouTubeScheduleTime == "" {
			return fmt.Errorf("YOUTUBE_SCHEDULE_TIME%s обязателен при YOUTUBE_ENABLED=true", suffix)
		}
	}

	return nil
}

func appendLangDir(base, lang string) string {
	if filepath.Base(filepath.Clean(base)) == lang {
		return base
	}
	return filepath.Join(base, lang)
}

func envWithLang(name, suffix string) string {
	return strings.TrimSpace(os.Getenv(name + suffix))
}

func (cfg *Config) LangStateDir() string {
	return appendLangDir(cfg.StateDir, cfg.VideoLang)
}
