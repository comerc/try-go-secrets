package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type TTSUsageState struct {
	path  string
	Daily map[string]int `json:"daily"` // "YYYY-MM-DD" -> кол-во символов
}

func LoadTTSUsage(stateDir string) (*TTSUsageState, error) {
	path := filepath.Join(stateDir, "tts_usage.json")
	s := &TTSUsageState{
		path:  path,
		Daily: make(map[string]int),
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *TTSUsageState) TodayUsage() int {
	return s.Daily[today()]
}

func (s *TTSUsageState) Add(chars int) error {
	s.Daily[today()] += chars
	return s.save()
}

func (s *TTSUsageState) save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func today() string {
	return time.Now().Format("2006-01-02")
}
