package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type YouTubeScheduleEntry struct {
	ID   string `json:"id"`
	Date string `json:"date"`
}

type YouTubeScheduleState struct {
	path   string
	byDate map[string]YouTubeScheduleEntry
	byID   map[string]YouTubeScheduleEntry
}

func LoadYouTubeSchedule(stateDir string) (*YouTubeScheduleState, error) {
	path := filepath.Join(stateDir, "youtube_schedule.json")
	s := &YouTubeScheduleState{
		path:   path,
		byDate: make(map[string]YouTubeScheduleEntry),
		byID:   make(map[string]YouTubeScheduleEntry),
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) || len(data) == 0 {
		return s, nil
	}
	if err != nil {
		return nil, err
	}

	var entries []YouTubeScheduleEntry
	if err := json.Unmarshal(data, &entries); err == nil {
		for _, e := range entries {
			s.addLoaded(e)
		}
		return s, nil
	}

	return nil, fmt.Errorf("ошибка чтения youtube_schedule.json: ожидается JSON-массив")
}

func (s *YouTubeScheduleState) addLoaded(entry YouTubeScheduleEntry) {
	if entry.ID != "" && entry.Date != "" {
		s.byDate[entry.Date] = entry
		s.byID[entry.ID] = entry
	}
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
		if existing.Date == date {
			return nil
		}
		return fmt.Errorf("id %s already scheduled for %s", id, existing.Date)
	}
	if existing, ok := s.byDate[date]; ok {
		return fmt.Errorf("date %s already reserved by %s", date, existing.ID)
	}
	entry := YouTubeScheduleEntry{ID: id, Date: date}
	s.byDate[date] = entry
	s.byID[id] = entry
	return s.save()
}

func (s *YouTubeScheduleState) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}

	dates := make([]string, 0, len(s.byDate))
	for date := range s.byDate {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	out := make([]YouTubeScheduleEntry, 0, len(dates))
	for _, date := range dates {
		out = append(out, s.byDate[date])
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
