package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go-secrets-pipeline/pkg/config"
	"go-secrets-pipeline/pkg/models"
)

type VideoService struct {
	cfg    *config.Config
	client *http.Client
}

// SubtitleWord — слово с временной меткой начала произношения (в секундах)
type SubtitleWord struct {
	Word     string  `json:"word"`
	StartSec float64 `json:"startSec"`
}

type renderRequest struct {
	Code          string         `json:"code"`
	Lang          string         `json:"lang"`
	Slug          string         `json:"slug"`
	OutputPath    string         `json:"outputPath"`
	Width         int            `json:"width"`
	Height        int            `json:"height"`
	FPS           int            `json:"fps"`
	AudioDuration float64        `json:"audioDuration"`
	SubtitleWords []SubtitleWord `json:"subtitleWords"`
}

type renderResponse struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"outputPath"`
	Error      string `json:"error"`
}

func NewVideoService(cfg *config.Config) *VideoService {
	return &VideoService{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

// RenderCodeVideo запрашивает у Puppeteer-сервиса видео с анимацией кода
func (s *VideoService) RenderCodeVideo(spec *models.VideoSpec, code, lang string, audioDuration float64, subtitleWords []SubtitleWord) (string, error) {
	rawVideoPath := filepath.Join(
		s.cfg.OutputDir, "videos", "raw", spec.Slug+"-code.mp4",
	)
	if err := os.MkdirAll(filepath.Dir(rawVideoPath), 0755); err != nil {
		return "", err
	}

	reqBody := renderRequest{
		Code:          code,
		Lang:          lang,
		Slug:          spec.Slug,
		OutputPath:    rawVideoPath,
		Width:         spec.Width,
		Height:        spec.Height,
		FPS:           spec.FPS,
		AudioDuration: audioDuration,
		SubtitleWords: subtitleWords,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := s.client.Post(s.cfg.PuppeteerURL+"/render", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("Puppeteer запрос: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Puppeteer ошибка %d: %s", resp.StatusCode, string(body))
	}

	var renderResp renderResponse
	if err := json.Unmarshal(body, &renderResp); err != nil {
		return "", err
	}
	if !renderResp.Success {
		return "", fmt.Errorf("Puppeteer render: %s", renderResp.Error)
	}

	return rawVideoPath, nil
}

// MergeAudioVideo объединяет аудио и видео через FFmpeg
func (s *VideoService) MergeAudioVideo(videoPath, audioPath, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	// FFmpeg: video + audio → финальный MP4
	// -shortest обрезает по более короткому треку
	cmd := exec.Command("ffmpeg", "-y",
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy",
		"-c:a", "libmp3lame", // AAC отключён в VSCode Electron; MP3 работает
		"-b:a", "128k",
		"-ar", "44100",
		"-shortest",
		"-movflags", "+faststart",
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("FFmpeg merge: %w\n%s", err, stderr.String())
	}

	return nil
}

// GetDuration возвращает длительность видео в секундах через ffprobe
func (s *VideoService) GetDuration(videoPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		videoPath,
	)

	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}

	var dur float64
	if _, err := fmt.Sscanf(string(out), "%f", &dur); err != nil {
		return 0, fmt.Errorf("парсинг длительности: %w", err)
	}
	return dur, nil
}

// WaitForPuppeteer ожидает готовности Puppeteer сервиса
func (s *VideoService) WaitForPuppeteer(maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		resp, err := s.client.Get(s.cfg.PuppeteerURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("Puppeteer сервис недоступен через %s", maxWait)
}
