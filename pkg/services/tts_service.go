package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aka/semarang/pkg/state"
)

// TTSService handles interactions with Yandex SpeechKit
type TTSService struct {
	apiKey     string
	folderID   string
	baseURL    string
	voice      string
	lang       string
	httpClient *http.Client
	usageMgr   *state.TTSUsageManager
	maxChars   int
}

// NewTTSService creates a new TTSService
func NewTTSService(apiKey, folderID, baseURL, voice, lang string, usageMgr *state.TTSUsageManager, maxChars int) *TTSService {
	return &TTSService{
		apiKey:   apiKey,
		folderID: folderID,
		baseURL:  baseURL,
		voice:    voice,
		lang:     lang,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		usageMgr: usageMgr,
		maxChars: maxChars,
	}
}

// Synthesize converts text to speech and saves to file
// Returns the duration in milliseconds
func (s *TTSService) Synthesize(text, outputPath string) (int, error) {
	// Check text length
	if len(text) > s.maxChars {
		return 0, fmt.Errorf("text exceeds max length: %d chars (max: %d)", len(text), s.maxChars)
	}

	// Check quota
	if s.usageMgr.IsQuotaExceeded() {
		return 0, fmt.Errorf("TTS quota exceeded: %d/%d chars used",
			s.usageMgr.GetCharsUsed(), s.maxChars)
	}

	// Build request URL
	params := url.Values{}
	params.Set("text", text)
	params.Set("voice", s.voice)
	params.Set("lang", s.lang)
	params.Set("format", "mp3")

	reqURL := s.baseURL + "?" + params.Encode()

	// Create request
	req, err := http.NewRequest("POST", reqURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Api-Key "+s.apiKey)

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("TTS API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Create output directory
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return 0, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save audio file
	file, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return 0, fmt.Errorf("failed to save audio: %w", err)
	}

	// Get audio duration using ffprobe
	durationMs, err := s.getAudioDuration(outputPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get audio duration: %w", err)
	}

	// Update usage
	if err := s.usageMgr.AddUsage(len(text)); err != nil {
		return 0, fmt.Errorf("failed to update TTS usage: %w", err)
	}

	return durationMs, nil
}

// getAudioDuration gets the duration of an audio file using ffprobe
func (s *TTSService) getAudioDuration(audioPath string) (int, error) {
	// Try ffprobe first
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "json",
		audioPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to estimation (8 chars/sec for Russian)
		return s.estimateDuration(audioPath), nil
	}

	var result struct {
		Format struct {
			Duration float64 `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return s.estimateDuration(audioPath), nil
	}

	return int(result.Format.Duration * 1000), nil
}

// estimateDuration estimates duration based on file size
// Fallback when ffprobe is not available
func (s *TTSService) estimateDuration(audioPath string) int {
	// Rough estimation: 1KB ≈ 0.1 seconds at 128kbps MP3
	info, err := os.Stat(audioPath)
	if err != nil {
		return 5000 // 5 seconds default
	}

	// Duration in seconds ≈ file_size / 12800 (bits per second)
	durationSec := float64(info.Size()) / 12800.0
	return int(durationSec * 1000)
}

// EstimateDurationFromText estimates speech duration from text
func (s *TTSService) EstimateDurationFromText(text string, charsPerSec float64) int {
	durationSec := float64(len(text)) / charsPerSec
	return int(durationSec * 1000) // milliseconds
}

// GetQuotaInfo returns current quota information
func (s *TTSService) GetQuotaInfo() (used, remaining, total int) {
	return s.usageMgr.GetCharsUsed(),
		s.usageMgr.GetCharsRemaining(),
		s.maxChars
}

// GetCallsToday returns the number of TTS calls today
func (s *TTSService) GetCallsToday() int {
	return s.usageMgr.GetCallsToday()
}
