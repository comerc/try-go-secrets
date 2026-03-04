package state

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aka/semarang/pkg/models"
)

// ProcessedManager manages the processed files state
type ProcessedManager struct {
	statePath string
	state    *models.ProcessedState
}

// NewProcessedManager creates a new ProcessedManager
func NewProcessedManager(statePath string) (*ProcessedManager, error) {
	pm := &ProcessedManager{
		statePath: statePath,
	}

	// Load existing state or create new
	if err := pm.load(); err != nil {
		return nil, fmt.Errorf("failed to load processed state: %w", err)
	}

	return pm, nil
}

// load loads the state from file
func (pm *ProcessedManager) load() error {
	data, err := os.ReadFile(pm.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new state
			pm.state = models.NewProcessedState()
			return pm.save()
		}
		return err
	}

	// Parse existing state
	if err := json.Unmarshal(data, &pm.state); err != nil {
		// Invalid format, create new state
		pm.state = models.NewProcessedState()
		return pm.save()
	}

	return nil
}

// save saves the state to file
func (pm *ProcessedManager) save() error {
	data, err := json.MarshalIndent(pm.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(pm.statePath, data, 0644)
}

// IsProcessed checks if a file has been processed
func (pm *ProcessedManager) IsProcessed(filename string) bool {
	if pm.state == nil {
		return false
	}
	return pm.state.IsProcessed(filename)
}

// MarkProcessed marks a file as processed
func (pm *ProcessedManager) MarkProcessed(filename, date, videoPath, status string) error {
	if pm.state == nil {
		return fmt.Errorf("state not initialized")
	}

	pm.state.AddProcessedFile(filename, date, videoPath, status)
	return pm.save()
}

// GetProcessedFiles returns a map of all processed filenames
func (pm *ProcessedManager) GetProcessedFiles() map[string]bool {
	if pm.state == nil {
		return make(map[string]bool)
	}
	return pm.state.GetProcessedFiles()
}

// GetAllEntries returns all processed file entries
func (pm *ProcessedManager) GetAllEntries() []models.ProcessedFile {
	if pm.state == nil {
		return []models.ProcessedFile{}
	}
	return pm.state.ProcessedFiles
}

// GetLastUpdated returns the last updated timestamp
func (pm *ProcessedManager) GetLastUpdated() string {
	if pm.state == nil {
		return ""
	}
	return pm.state.LastUpdated
}

// GetCount returns the count of processed files
func (pm *ProcessedManager) GetCount() int {
	if pm.state == nil {
		return 0
	}
	return len(pm.state.ProcessedFiles)
}
