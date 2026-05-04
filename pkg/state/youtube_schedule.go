package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type YouTubeScheduleState struct {
	path   string
	byDate map[string]string
	byID   map[string]string
}

func LoadYouTubeSchedule(stateDir string) (*YouTubeScheduleState, error) {
	path := filepath.Join(stateDir, "youtube_schedule.json")
	s := &YouTubeScheduleState{
		path:   path,
		byDate: make(map[string]string),
		byID:   make(map[string]string),
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) || len(data) == 0 {
		return s, nil
	}
	if err != nil {
		return nil, err
	}

	var entries map[string]string
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("ошибка чтения youtube_schedule.json: ожидается JSON-объект id -> date")
	}
	for id, date := range entries {
		if id != "" && date != "" {
			s.byID[id] = date
			s.byDate[date] = id
		}
	}
	return s, nil
}

func FormatVideoID(fileNum int) string {
	return fmt.Sprintf("%03d", fileNum)
}

func (s *YouTubeScheduleState) IsReserved(date string) bool {
	_, ok := s.byDate[date]
	return ok
}

func (s *YouTubeScheduleState) IsPublished(id string) bool {
	_, ok := s.byID[id]
	return ok
}

func (s *YouTubeScheduleState) Reserve(id, date string) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}
	if date == "" {
		return fmt.Errorf("date is empty")
	}
	if existing, ok := s.byID[id]; ok {
		if existing == date {
			return nil
		}
		return fmt.Errorf("id %s already scheduled for %s", id, existing)
	}
	if existing, ok := s.byDate[date]; ok {
		return fmt.Errorf("date %s already reserved by %s", date, existing)
	}
	s.byDate[date] = id
	s.byID[id] = date
	return s.save()
}

func (s *YouTubeScheduleState) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.byID, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
