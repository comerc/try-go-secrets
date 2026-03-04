package state

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aka/semarang/pkg/models"
)

// TTSUsageManager manages the TTS usage state
type TTSUsageManager struct {
	statePath  string
	state      *models.TTSUsage
	maxChars   int
}

// NewTTSUsageManager creates a new TTSUsageManager
func NewTTSUsageManager(statePath string, maxChars int) (*TTSUsageManager, error) {
	tum := &TTSUsageManager{
		statePath: statePath,
		maxChars:  maxChars,
	}

	// Load existing state or create new
	if err := tum.load(); err != nil {
		return nil, fmt.Errorf("failed to load TTS usage state: %w", err)
	}

	return tum, nil
}

// load loads the state from file
func (tum *TTSUsageManager) load() error {
	data, err := os.ReadFile(tum.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new state
			tum.state = models.NewTTSUsage(tum.maxChars)
			return tum.save()
		}
		return err
	}

	// Parse existing state
	if err := json.Unmarshal(data, &tum.state); err != nil {
		// Invalid format, create new state
		tum.state = models.NewTTSUsage(tum.maxChars)
		return tum.save()
	}

	// Check if it's a new day
	if tum.state.IsNewDay() {
		tum.state.ResetForNewDay(tum.maxChars)
		return tum.save()
	}

	return nil
}

// save saves the state to file
func (tum *TTSUsageManager) save() error {
	data, err := json.MarshalIndent(tum.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tum.statePath, data, 0644)
}

// AddUsage adds character usage to the TTS tracker
func (tum *TTSUsageManager) AddUsage(chars int) error {
	if tum.state == nil {
		return fmt.Errorf("state not initialized")
	}

	// Check for new day
	if tum.state.IsNewDay() {
		tum.state.ResetForNewDay(tum.maxChars)
	}

	// Add usage
	if err := tum.state.AddUsage(chars); err != nil {
		return tum.save()
	}

	return tum.save()
}

// GetCharsRemaining returns the number of characters remaining for today
func (tum *TTSUsageManager) GetCharsRemaining() int {
	if tum.state == nil {
		return tum.maxChars
	}
	return tum.state.CharsRemaining
}

// GetCharsUsed returns the number of characters used today
func (tum *TTSUsageManager) GetCharsUsed() int {
	if tum.state == nil {
		return 0
	}
	return tum.state.CharsUsed
}

// GetCallsToday returns the number of TTS calls today
func (tum *TTSUsageManager) GetCallsToday() int {
	if tum.state == nil {
		return 0
	}
	return tum.state.CallsToday
}

// GetLastUpdated returns the last updated timestamp
func (tum *TTSUsageManager) GetLastUpdated() string {
	if tum.state == nil {
		return ""
	}
	return tum.state.LastUpdated
}

// IsQuotaExceeded checks if the TTS quota is exceeded
func (tum *TTSUsageManager) IsQuotaExceeded() bool {
	if tum.state == nil {
		return false
	}
	return tum.state.CharsRemaining <= 0
}

// GetDate returns the current date of the usage tracker
func (tum *TTSUsageManager) GetDate() string {
	if tum.state == nil {
		return time.Now().Format("2006-01-02")
	}
	return tum.state.Date
}

// ResetDaily resets the usage for a new day
func (tum *TTSUsageManager) ResetDaily() error {
	if tum.state == nil {
		return fmt.Errorf("state not initialized")
	}

	tum.state.ResetForNewDay(tum.maxChars)
	return tum.save()
}
