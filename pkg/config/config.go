package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config holds all configuration for the semarang pipeline
type Config struct {
	// z.ai API
	ZAIAPIKey     string
	ZAIAPIURL     string

	// Yandex SpeechKit
	YandexAPIKey     string
	YandexFolderID   string
	YandexTTSURL     string
	YandexVoice      string
	YandexLanguage   string

	// TTS settings
	TTSMaxCharsPerDay        int
	TTSSpeechRateCharsPerSec float64

	// Video settings
	VideoMaxDurationSeconds  int
	VideoTargetDurationSeconds int
	VideoWidth  int
	VideoHeight int

	// Puppeteer service
	PuppeteerServiceURL string

	// Paths
	RawDir    string
	OutputDir string
	StateDir  string

	// Logging
	LogLevel string
}

// New creates a new Config from environment variables
func New() (*Config, error) {
	cfg := &Config{
		// z.ai API
		ZAIAPIKey:     getEnv("ZAI_API_KEY", ""),
		ZAIAPIURL:     getEnv("ZAI_API_URL", "https://api.z.ai/v1"),

		// Yandex SpeechKit
		YandexAPIKey:     getEnv("YANDEX_API_KEY", ""),
		YandexFolderID:   getEnv("YANDEX_FOLDER_ID", ""),
		YandexTTSURL:     getEnv("YANDEX_TTS_URL", "https://stt.api.cloud.yandex.net/speech/v1/tts:synthesize"),
		YandexVoice:      getEnv("YANDEX_VOICE", "alice"),
		YandexLanguage:   getEnv("YANDEX_LANGUAGE", "ru-RU"),

		// TTS settings
		TTSMaxCharsPerDay:        getEnvInt("TTS_MAX_CHARS_PER_DAY", 2000),
		TTSSpeechRateCharsPerSec: getEnvFloat("TTS_SPEECH_RATE_CHARS_PER_SEC", 8.0),

		// Video settings
		VideoMaxDurationSeconds:  getEnvInt("VIDEO_MAX_DURATION_SECONDS", 60),
		VideoTargetDurationSeconds: getEnvInt("VIDEO_TARGET_DURATION_SECONDS", 50),
		VideoWidth:  getEnvInt("VIDEO_WIDTH", 1080),
		VideoHeight: getEnvInt("VIDEO_HEIGHT", 1920),

		// Puppeteer service
		PuppeteerServiceURL: getEnv("PUPPETEER_SERVICE_URL", "http://puppeteer:3000"),

		// Paths
		RawDir:    getEnv("RAW_DIR", "/app/raw"),
		OutputDir: getEnv("OUTPUT_DIR", "/app/output"),
		StateDir:  getEnv("STATE_DIR", "/app/state"),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "INFO"),
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Ensure directories exist
	if err := cfg.EnsureDirs(); err != nil {
		return nil, fmt.Errorf("failed to ensure directories: %w", err)
	}

	return cfg, nil
}

// Validate checks that all required configuration is present
func (c *Config) Validate() error {
	if c.ZAIAPIKey == "" {
		return fmt.Errorf("ZAI_API_KEY is required")
	}
	if c.YandexAPIKey == "" {
		return fmt.Errorf("YANDEX_API_KEY is required")
	}
	if c.YandexFolderID == "" {
		return fmt.Errorf("YANDEX_FOLDER_ID is required")
	}
	if c.TTSMaxCharsPerDay <= 0 {
		return fmt.Errorf("TTS_MAX_CHARS_PER_DAY must be positive")
	}
	if c.TTSSpeechRateCharsPerSec <= 0 {
		return fmt.Errorf("TTS_SPEECH_RATE_CHARS_PER_SEC must be positive")
	}
	if c.VideoMaxDurationSeconds <= 0 {
		return fmt.Errorf("VIDEO_MAX_DURATION_SECONDS must be positive")
	}
	if c.VideoTargetDurationSeconds <= 0 || c.VideoTargetDurationSeconds > c.VideoMaxDurationSeconds {
		return fmt.Errorf("VIDEO_TARGET_DURATION_SECONDS must be between 1 and VIDEO_MAX_DURATION_SECONDS")
	}
	if c.VideoWidth <= 0 || c.VideoHeight <= 0 {
		return fmt.Errorf("VIDEO_WIDTH and VIDEO_HEIGHT must be positive")
	}
	return nil
}

// EnsureDirs creates all necessary directories if they don't exist
func (c *Config) EnsureDirs() error {
	dirs := []string{
		c.RawDir,
		filepath.Join(c.OutputDir, "scripts"),
		filepath.Join(c.OutputDir, "audio"),
		filepath.Join(c.OutputDir, "videos"),
		filepath.Join(c.OutputDir, "logs"),
		c.StateDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// GetOutputScriptsDir returns the directory for script files
func (c *Config) GetOutputScriptsDir() string {
	return filepath.Join(c.OutputDir, "scripts")
}

// GetOutputAudioDir returns the directory for audio files
func (c *Config) GetOutputAudioDir() string {
	return filepath.Join(c.OutputDir, "audio")
}

// GetOutputVideosDir returns the directory for video files
func (c *Config) GetOutputVideosDir() string {
	return filepath.Join(c.OutputDir, "videos")
}

// GetOutputLogsDir returns the directory for log files
func (c *Config) GetOutputLogsDir() string {
	return filepath.Join(c.OutputDir, "logs")
}

// GetProcessedStateFile returns the path to processed.json
func (c *Config) GetProcessedStateFile() string {
	return filepath.Join(c.StateDir, "processed.json")
}

// GetTTSUsageStateFile returns the path to tts_usage.json
func (c *Config) GetTTSUsageStateFile() string {
	return filepath.Join(c.StateDir, "tts_usage.json")
}

// GetRawDir returns the directory with raw markdown files
func (c *Config) GetRawDir() string {
	return c.RawDir
}

// GetTodayString returns today's date in YYYY-MM-DD format
func (c *Config) GetTodayString() string {
	return time.Now().Format("2006-01-02")
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves an environment variable as int or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvFloat retrieves an environment variable as float64 or returns a default value
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}
