package models

import (
	"fmt"
	"time"
)

// ProcessedState represents the state of processed files
type ProcessedState struct {
	ProcessedFiles []ProcessedFile `json:"processed_files"`
	LastUpdated    string          `json:"last_updated"`
}

// ProcessedFile represents a single processed file
type ProcessedFile struct {
	File       string `json:"file"`
	Date       string `json:"date"`       // YYYY-MM-DD
	VideoPath  string `json:"video_path"`
	Status     string `json:"status"`     // "completed", "failed"
	CreatedAt  string `json:"created_at"` // ISO 8601
}

// TTSUsage represents TTS usage tracking
type TTSUsage struct {
	Date            string `json:"date"`            // YYYY-MM-DD
	CharsUsed       int    `json:"chars_used"`
	CharsRemaining  int    `json:"chars_remaining"`
	CallsToday      int    `json:"calls_today"`
	LastUpdated     string `json:"last_updated"`
}

// NewProcessedState creates a new ProcessedState
func NewProcessedState() *ProcessedState {
	return &ProcessedState{
		ProcessedFiles: make([]ProcessedFile, 0),
		LastUpdated:    time.Now().Format(time.RFC3339),
	}
}

// AddProcessedFile adds a processed file to the state
func (ps *ProcessedState) AddProcessedFile(file, date, videoPath, status string) {
	ps.ProcessedFiles = append(ps.ProcessedFiles, ProcessedFile{
		File:      file,
		Date:      date,
		VideoPath: videoPath,
		Status:    status,
		CreatedAt: time.Now().Format(time.RFC3339),
	})
	ps.LastUpdated = time.Now().Format(time.RFC3339)
}

// IsProcessed checks if a file has been processed
func (ps *ProcessedState) IsProcessed(filename string) bool {
	for _, pf := range ps.ProcessedFiles {
		if pf.File == filename {
			return pf.Status == "completed"
		}
	}
	return false
}

// GetProcessedFiles returns a set of processed filenames
func (ps *ProcessedState) GetProcessedFiles() map[string]bool {
	files := make(map[string]bool)
	for _, pf := range ps.ProcessedFiles {
		if pf.Status == "completed" {
			files[pf.File] = true
		}
	}
	return files
}

// NewTTSUsage creates a new TTSUsage for today
func NewTTSUsage(maxChars int) *TTSUsage {
	today := time.Now().Format("2006-01-02")
	return &TTSUsage{
		Date:           today,
		CharsUsed:      0,
		CharsRemaining:  maxChars,
		CallsToday:     0,
		LastUpdated:    time.Now().Format(time.RFC3339),
	}
}

// AddUsage adds character usage to the TTS tracker
func (tu *TTSUsage) AddUsage(chars int) error {
	// Check if it's a new day
	today := time.Now().Format("2006-01-02")
	if tu.Date != today {
		// Reset for new day (this is handled by state management)
		tu.Date = today
		tu.CharsUsed = 0
		tu.CallsToday = 0
	}

	if tu.CharsRemaining < chars {
		return fmt.Errorf("not enough TTS quota: need %d chars, have %d remaining",
			chars, tu.CharsRemaining)
	}

	tu.CharsUsed += chars
	tu.CharsRemaining -= chars
	tu.CallsToday++
	tu.LastUpdated = time.Now().Format(time.RFC3339)
	return nil
}

// IsNewDay checks if the TTS usage is for a different day
func (tu *TTSUsage) IsNewDay() bool {
	today := time.Now().Format("2006-01-02")
	return tu.Date != today
}

// ResetForNewDay resets the usage for a new day
func (tu *TTSUsage) ResetForNewDay(maxChars int) {
	today := time.Now().Format("2006-01-02")
	tu.Date = today
	tu.CharsUsed = 0
	tu.CharsRemaining = maxChars
	tu.CallsToday = 0
	tu.LastUpdated = time.Now().Format(time.RFC3339)
}
