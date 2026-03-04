package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// QualityChecker validates generated videos
type QualityChecker struct {
	maxDurationMs  int
	maxSizeBytes   int64
}

// CheckResult represents the result of a quality check
type CheckResult struct {
	Passed  bool     `json:"passed"`
	VideoPath string `json:"video_path"`
	Issues  []Issue `json:"issues"`
}

// Issue represents a quality issue
type Issue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Severity string `json:"severity"` // "error", "warning"
}

// NewQualityChecker creates a new QualityChecker
func NewQualityChecker(maxDurationSeconds int, maxSizeMB int64) *QualityChecker {
	return &QualityChecker{
		maxDurationMs: maxDurationSeconds * 1000,
		maxSizeBytes:  maxSizeMB * 1024 * 1024,
	}
}

// Check performs quality checks on a video file
func (qc *QualityChecker) Check(videoPath string) (*CheckResult, error) {
	issues := make([]Issue, 0)

	// Check if file exists
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("video file does not exist: %s", videoPath)
	}

	// Get file info
	info, err := os.Stat(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check file size
	if info.Size() == 0 {
		issues = append(issues, Issue{
			Code:    "empty_file",
			Message: "Video file is empty",
			Severity: "error",
		})
	} else if info.Size() > qc.maxSizeBytes {
		issues = append(issues, Issue{
			Code:    "file_too_large",
			Message: fmt.Sprintf("Video file too large: %d bytes (max: %d bytes)", info.Size(), qc.maxSizeBytes),
			Severity: "warning",
		})
	}

	// Check file extension
	if filepath.Ext(videoPath) != ".mp4" {
		issues = append(issues, Issue{
			Code:    "invalid_format",
			Message: fmt.Sprintf("Invalid file format: %s (expected .mp4)", filepath.Ext(videoPath)),
			Severity: "error",
		})
	}

	// Get video duration using ffprobe
	durationMs, err := qc.getVideoDuration(videoPath)
	if err != nil {
		issues = append(issues, Issue{
			Code:    "duration_check_failed",
			Message: fmt.Sprintf("Failed to get video duration: %v", err),
			Severity: "error",
		})
	} else {
		if durationMs == 0 {
			issues = append(issues, Issue{
				Code:    "zero_duration",
				Message: "Video has zero duration",
				Severity: "error",
			})
		} else if durationMs > qc.maxDurationMs {
			issues = append(issues, Issue{
				Code:    "duration_exceeded",
				Message: fmt.Sprintf("Video duration exceeds limit: %dms (max: %dms)", durationMs, qc.maxDurationMs),
				Severity: "error",
			})
		}
	}

	// Check video codec (if ffprobe available)
	codecIssue := qc.checkVideoCodec(videoPath)
	if codecIssue != nil {
		issues = append(issues, *codecIssue)
	}

	// Check audio codec (if ffprobe available)
	audioIssue := qc.checkAudioCodec(videoPath)
	if audioIssue != nil {
		issues = append(issues, *audioIssue)
	}

	// Determine if passed
	passed := true
	for _, issue := range issues {
		if issue.Severity == "error" {
			passed = false
			break
		}
	}

	result := &CheckResult{
		Passed:   passed,
		VideoPath: videoPath,
		Issues:   issues,
	}

	// Log results
	qc.logResults(result)

	return result, nil
}

// getVideoDuration returns the duration of a video file in milliseconds
func (qc *QualityChecker) getVideoDuration(videoPath string) (int, error) {
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

// checkVideoCodec checks if the video codec is valid
func (qc *QualityChecker) checkVideoCodec(videoPath string) *Issue {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name",
		"-of", "json",
		videoPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	var result struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil
	}

	if len(result.Streams) == 0 {
		return &Issue{
			Code:    "no_video_stream",
			Message: "No video stream found",
			Severity: "error",
		}
	}

	codec := result.Streams[0].CodecName
	if codec != "h264" {
		return &Issue{
			Code:    "invalid_codec",
			Message: fmt.Sprintf("Invalid video codec: %s (expected: h264)", codec),
			Severity: "warning",
		}
	}

	return nil
}

// checkAudioCodec checks if the audio codec is valid
func (qc *QualityChecker) checkAudioCodec(videoPath string) *Issue {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_name",
		"-of", "json",
		videoPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// No audio stream might be okay for some videos
		return nil
	}

	var result struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil
	}

	if len(result.Streams) == 0 {
		return &Issue{
			Code:    "no_audio_stream",
			Message: "No audio stream found",
			Severity: "warning",
		}
	}

	codec := result.Streams[0].CodecName
	if codec != "aac" {
		return &Issue{
			Code:    "invalid_audio_codec",
			Message: fmt.Sprintf("Invalid audio codec: %s (expected: aac)", codec),
			Severity: "warning",
		}
	}

	return nil
}

// logResults logs the quality check results
func (qc *QualityChecker) logResults(result *CheckResult) {
	fmt.Printf("[QualityChecker] Checking: %s\n", result.VideoPath)

	if result.Passed {
		fmt.Printf("[QualityChecker] ✓ PASSED\n")
	} else {
		fmt.Printf("[QualityChecker] ✗ FAILED\n")
	}

	for _, issue := range result.Issues {
		if issue.Severity == "error" {
			fmt.Printf("[QualityChecker] ✗ ERROR: %s - %s\n", issue.Code, issue.Message)
		} else {
			fmt.Printf("[QualityChecker] ⚠ WARNING: %s - %s\n", issue.Code, issue.Message)
		}
	}
}

// GetDetailedReport returns a detailed report of the quality check
func (qc *QualityChecker) GetDetailedReport(result *CheckResult) string {
	report := fmt.Sprintf("Quality Check Report for: %s\n", result.VideoPath)
	report += fmt.Sprintf("Status: %s\n\n", map[bool]string{true: "PASSED", false: "FAILED"}[result.Passed])

	if len(result.Issues) == 0 {
		report += "No issues found.\n"
	} else {
		report += "Issues:\n"
		for _, issue := range result.Issues {
			report += fmt.Sprintf("  [%s] %s: %s\n", issue.Severity, issue.Code, issue.Message)
		}
	}

	return report
}
