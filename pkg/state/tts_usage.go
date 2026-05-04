package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type TTSUsageEntry struct {
	Runs  int
	Chars int
}

type TTSUsageState struct {
	path  string
	Daily map[string]TTSUsageEntry // "YYYY-MM-DD" -> запуски и кол-во символов
}

func LoadTTSUsage(stateDir string) (*TTSUsageState, error) {
	path := filepath.Join(stateDir, "tts_usage.json")
	s := &TTSUsageState{
		path:  path,
		Daily: make(map[string]TTSUsageEntry),
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}

	if err := s.unmarshal(data); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *TTSUsageState) TodayUsage() TTSUsageEntry {
	return s.Daily[today()]
}

func (s *TTSUsageState) Add(chars int) (TTSUsageEntry, error) {
	day := today()
	entry := s.Daily[day]
	entry.Runs++
	entry.Chars += chars
	s.Daily[day] = entry
	return entry, s.save()
}

func (s *TTSUsageState) unmarshal(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for day, value := range raw {
		entry, err := parseEntry(value)
		if err != nil {
			return fmt.Errorf("%s: %w", day, err)
		}
		s.Daily[day] = entry
	}
	return nil
}

func (s *TTSUsageState) save() error {
	out := make(map[string]string, len(s.Daily))
	for day, entry := range s.Daily {
		out[day] = fmt.Sprintf("%d-%d", entry.Runs, entry.Chars)
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func parseEntry(value any) (TTSUsageEntry, error) {
	switch v := value.(type) {
	case string:
		parts := strings.Split(v, "-")
		if len(parts) != 2 {
			return TTSUsageEntry{}, fmt.Errorf("ожидается формат runs-chars")
		}
		runs, err := strconv.Atoi(parts[0])
		if err != nil {
			return TTSUsageEntry{}, fmt.Errorf("runs %q: %w", parts[0], err)
		}
		chars, err := strconv.Atoi(parts[1])
		if err != nil {
			return TTSUsageEntry{}, fmt.Errorf("chars %q: %w", parts[1], err)
		}
		return TTSUsageEntry{Runs: runs, Chars: chars}, nil
	default:
		return TTSUsageEntry{}, fmt.Errorf("ожидается строка runs-chars")
	}
}

func today() string {
	return time.Now().Format("2006-01-02")
}
