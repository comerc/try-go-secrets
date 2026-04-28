package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ProcessedEntry struct {
	FileNum   int    `json:"file_num"`
	Slug      string `json:"slug"`
	VideoPath string `json:"video_path"`
	Date      string `json:"date"`
}

type ProcessedState struct {
	path    string
	Entries []ProcessedEntry `json:"entries"`
}

func LoadProcessed(stateDir string) (*ProcessedState, error) {
	path := filepath.Join(stateDir, "processed.json")
	ps := &ProcessedState{path: path}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ps, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, ps); err != nil {
		return nil, fmt.Errorf("ошибка чтения processed.json: %w", err)
	}
	return ps, nil
}

func (ps *ProcessedState) IsProcessed(fileNum int) bool {
	for _, e := range ps.Entries {
		if e.FileNum == fileNum {
			return true
		}
	}
	return false
}

func (ps *ProcessedState) Add(fileNum int, slug, videoPath string) error {
	ps.Entries = append(ps.Entries, ProcessedEntry{
		FileNum:   fileNum,
		Slug:      slug,
		VideoPath: videoPath,
		Date:      time.Now().Format("2006-01-02"),
	})
	return ps.save()
}

func (ps *ProcessedState) save() error {
	data, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ps.path, data, 0644)
}
