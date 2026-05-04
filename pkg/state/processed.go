package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

type ProcessedEntry struct {
	FileNum   int    `json:"file_num"`
	VideoPath string `json:"video_path"`
	Date      string `json:"date"`
}

type ProcessedState struct {
	path string
	nums map[int]bool
}

func LoadProcessed(stateDir string) (*ProcessedState, error) {
	path := filepath.Join(stateDir, "processed.json")
	ps := &ProcessedState{path: path, nums: make(map[int]bool)}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ps, nil
	}
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return ps, nil
	}

	if err := ps.unmarshal(data); err != nil {
		return nil, fmt.Errorf("ошибка чтения processed.json: %w", err)
	}
	return ps, nil
}

func (ps *ProcessedState) IsProcessed(fileNum int) bool {
	return ps.nums[fileNum]
}

func (ps *ProcessedState) Add(fileNum int) error {
	ps.nums[fileNum] = true
	return ps.save()
}

func (ps *ProcessedState) unmarshal(data []byte) error {
	var nums []string
	if err := json.Unmarshal(data, &nums); err == nil {
		for _, s := range nums {
			n, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("номер %q: %w", s, err)
			}
			ps.nums[n] = true
		}
		return nil
	}

	var old struct {
		Entries []ProcessedEntry `json:"entries"`
	}
	if err := json.Unmarshal(data, &old); err != nil {
		return err
	}
	for _, e := range old.Entries {
		ps.nums[e.FileNum] = true
	}
	return nil
}

func (ps *ProcessedState) save() error {
	nums := make([]int, 0, len(ps.nums))
	for n := range ps.nums {
		nums = append(nums, n)
	}
	sort.Ints(nums)

	out := make([]string, 0, len(nums))
	for _, n := range nums {
		out = append(out, fmt.Sprintf("%03d", n))
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ps.path, data, 0644)
}
