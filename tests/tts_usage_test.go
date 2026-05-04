package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-secrets-pipeline/pkg/state"
)

func TestTTSUsageWritesRunsAndChars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tts_usage.json")

	usage, err := state.LoadTTSUsage(dir)
	if err != nil {
		t.Fatal(err)
	}

	entry, err := usage.Add(10)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Runs != 1 || entry.Chars != 10 {
		t.Fatalf("first entry = %+v, want runs=1 chars=10", entry)
	}

	entry, err = usage.Add(5)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Runs != 2 || entry.Chars != 15 {
		t.Fatalf("second entry = %+v, want runs=2 chars=15", entry)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var saved map[string]string
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	today := time.Now().Format("2006-01-02")
	if saved[today] != "2-15" {
		t.Fatalf("saved today = %q, want 2-15; data: %s", saved[today], data)
	}
}
