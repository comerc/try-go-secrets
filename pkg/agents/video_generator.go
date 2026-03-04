package agents

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aka/semarang/pkg/models"
	"github.com/aka/semarang/pkg/services"
)

// VideoGenerator handles generating videos from scripts
type VideoGenerator struct {
	ttsService     *services.TTSService
	videoService   *services.VideoService
	audioDir       string
	videosDir      string
}

// NewVideoGenerator creates a new VideoGenerator
func NewVideoGenerator(
	ttsService *services.TTSService,
	videoService *services.VideoService,
	audioDir, videosDir string,
) *VideoGenerator {
	return &VideoGenerator{
		ttsService:   ttsService,
		videoService: videoService,
		audioDir:     audioDir,
		videosDir:    videosDir,
	}
}

// Generate generates a video from a script
func (vg *VideoGenerator) Generate(script *models.Script) (string, error) {
	vg.logInfo("Starting video generation for: %s", script.Title)

	// Ensure directories exist
	if err := os.MkdirAll(vg.audioDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create audio directory: %w", err)
	}
	if err := os.MkdirAll(vg.videosDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create videos directory: %w", err)
	}

	// Generate TTS for voice segments
	if err := vg.generateTTS(script); err != nil {
		return "", fmt.Errorf("failed to generate TTS: %w", err)
	}

	// Generate video with audio
	videoPath, err := vg.videoService.GenerateVideoWithAudio(script, vg.audioDir, vg.videosDir)
	if err != nil {
		return "", fmt.Errorf("failed to generate video: %w", err)
	}

	vg.logInfo("Video generated: %s", videoPath)
	return videoPath, nil
}

// generateTTS generates TTS for all voice segments in the script
func (vg *VideoGenerator) generateTTS(script *models.Script) error {
	voiceSegments := script.GetVoiceSegments()

	if len(voiceSegments) == 0 {
		vg.logInfo("No voice segments to generate TTS for")
		return nil
	}

	vg.logInfo("Generating TTS for %d segment(s)", len(voiceSegments))

	// Check TTS quota
	used, remaining, total := vg.ttsService.GetQuotaInfo()
	vg.logInfo("TTS quota: %d/%d used, %d remaining", used, total, remaining)

	for i, seg := range voiceSegments {
		filename := fmt.Sprintf("%s_segment_%03d.mp3", script.Date, i)
		outputPath := filepath.Join(vg.audioDir, filename)

		vg.logInfo("Generating TTS for segment %d: %s", i, filename)

		durationMs, err := vg.ttsService.Synthesize(seg.Text, outputPath)
		if err != nil {
			return fmt.Errorf("failed to synthesize segment %d: %w", i, err)
		}

		// Update segment with TTS file and actual duration
		seg.TTSFile = filename
		seg.DurationMs = durationMs

		vg.logInfo("TTS segment %d generated: %s (%dms)", i, filename, durationMs)
	}

	// Recalculate total duration with actual TTS durations
	script.CalculateTotalDuration()
	vg.logInfo("Updated total duration: %dms", script.TotalDurationMs)

	return nil
}

// logInfo logs an info message
func (vg *VideoGenerator) logInfo(format string, args ...interface{}) {
	fmt.Printf("[VideoGenerator] "+format+"\n", args...)
}
