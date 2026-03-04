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

	"github.com/aka/semarang/pkg/models"
)

// VideoService handles video generation using Puppeteer service
type VideoService struct {
	puppeteerURL string
	width        int
	height       int
	httpClient   *http.Client
}

// NewVideoService creates a new VideoService
func NewVideoService(puppeteerURL string, width, height int) *VideoService {
	return &VideoService{
		puppeteerURL: puppeteerURL,
		width:        width,
		height:       height,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// GenerateVideo generates a video from a script
func (s *VideoService) GenerateVideo(script *models.Script, audioDir, videosDir string) (string, error) {
	var videoSegments []string

	// Generate segments
	for _, seg := range script.Segments {
		switch seg.Type {
		case "code":
			videoPath, err := s.generateCodeVideo(seg.Content, seg.DurationMs, videosDir)
			if err != nil {
				return "", fmt.Errorf("failed to generate code video: %w", err)
			}
			videoSegments = append(videoSegments, videoPath)
		case "diagram":
			videoPath, err := s.generateDiagramVideo(seg.Content, seg.DurationMs, videosDir)
			if err != nil {
				return "", fmt.Errorf("failed to generate diagram video: %w", err)
			}
			videoSegments = append(videoSegments, videoPath)
		case "voice":
			// Voice segments are handled separately via TTS
			// The TTS service will generate audio files
		}
	}

	// Combine all video segments
	outputPath := filepath.Join(videosDir, script.GetSlug()+".mp4")

	if len(videoSegments) == 0 {
		return "", fmt.Errorf("no video segments to combine")
	}

	if len(videoSegments) == 1 {
		// Just rename the single segment
		if err := os.Rename(videoSegments[0], outputPath); err != nil {
			return "", fmt.Errorf("failed to rename video: %w", err)
		}
		return outputPath, nil
	}

	// Combine multiple segments
	if err := s.combineVideos(videoSegments, outputPath); err != nil {
		return "", fmt.Errorf("failed to combine videos: %w", err)
	}

	return outputPath, nil
}

// GenerateVideoWithAudio generates a video and combines it with audio
func (s *VideoService) GenerateVideoWithAudio(script *models.Script, audioDir, videosDir string) (string, error) {
	var segmentFiles []string

	// Process each segment
	for _, seg := range script.Segments {
		switch seg.Type {
		case "voice":
			// Use TTS audio file
			audioPath := filepath.Join(audioDir, seg.TTSFile)
			if _, err := os.Stat(audioPath); err == nil {
				segmentFiles = append(segmentFiles, audioPath)
			} else {
				return "", fmt.Errorf("audio file not found: %s", audioPath)
			}
		case "code":
			videoPath, err := s.generateCodeVideo(seg.Content, seg.DurationMs, videosDir)
			if err != nil {
				return "", fmt.Errorf("failed to generate code video: %w", err)
			}
			segmentFiles = append(segmentFiles, videoPath)
		case "diagram":
			videoPath, err := s.generateDiagramVideo(seg.Content, seg.DurationMs, videosDir)
			if err != nil {
				return "", fmt.Errorf("failed to generate diagram video: %w", err)
			}
			segmentFiles = append(segmentFiles, videoPath)
		}
	}

	// Combine all segments into final video
	outputPath := filepath.Join(videosDir, fmt.Sprintf("%s-%s.mp4", script.Date, script.GetSlug()))

	if err := s.combineAllSegments(segmentFiles, outputPath); err != nil {
		return "", fmt.Errorf("failed to combine segments: %w", err)
	}

	return outputPath, nil
}

// generateCodeVideo generates a video of code animation
func (s *VideoService) generateCodeVideo(code string, durationMs int, outputDir string) (string, error) {
	outputPath := filepath.Join(outputDir, fmt.Sprintf("code-%d.mp4", time.Now().UnixNano()))

	reqBody := map[string]interface{}{
		"code":           code,
		"output_path":     outputPath,
		"duration_ms":     durationMs,
		"width":          s.width,
		"height":         s.height,
		"typing_speed_ms": 50,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.httpClient.Post(s.puppeteerURL+"/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		VideoPath  string `json:"video_path"`
		DurationMs int    `json:"duration_ms"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.VideoPath, nil
}

// generateDiagramVideo generates a video of a diagram animation
func (s *VideoService) generateDiagramVideo(diagram string, durationMs int, outputDir string) (string, error) {
	// For now, treat diagrams as code (can be enhanced later)
	return s.generateCodeVideo(diagram, durationMs, outputDir)
}

// combineVideos combines multiple video files into one
func (s *VideoService) combineVideos(inputPaths []string, outputPath string) error {
	// Build filter complex for concatenation
	filterComplex := ""
	for i := range inputPaths {
		filterComplex += fmt.Sprintf("[%d:v]scale=%d:%d[v%d];", i, s.width, s.height, i)
	}

	// Concatenate video streams
	concatInputs := ""
	for i := range inputPaths {
		if i > 0 {
			concatInputs += ","
		}
		concatInputs += fmt.Sprintf("v%d", i)
	}
	filterComplex += fmt.Sprintf("%sconcat=n=%d:v=1[outv]", concatInputs, len(inputPaths))

	// Build ffmpeg command
	args := []string{
		"-y", // Overwrite output file
	}

	// Add input files
	for _, path := range inputPaths {
		args = append(args, "-i", path)
	}

	args = append(args,
		"-filter_complex", filterComplex,
		"-map", "[outv]",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-r", "30",
		outputPath,
	)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// combineAllSegments combines video and audio segments
func (s *VideoService) combineAllSegments(segmentFiles []string, outputPath string) error {
	// This is a simplified version - real implementation would properly sync audio
	// For now, just concatenate all files as video
	return s.combineVideos(segmentFiles, outputPath)
}

// GetVideoDuration returns the duration of a video file in milliseconds
func (s *VideoService) GetVideoDuration(videoPath string) (int, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "json",
		videoPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result struct {
		Format struct {
			Duration float64 `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return 0, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	return int(result.Format.Duration * 1000), nil
}

// GetVideoSize returns the size of a video file in bytes
func (s *VideoService) GetVideoSize(videoPath string) (int64, error) {
	info, err := os.Stat(videoPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
