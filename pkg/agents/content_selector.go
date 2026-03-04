package agents

import (
	"fmt"
	"io/fs"
	"math/rand"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aka/semarang/pkg/state"
)

// ContentSelector handles selecting content files for processing
type ContentSelector struct {
	rawDir          string
	processedMgr    *state.ProcessedManager
	fileNumberRegex *regexp.Regexp
}

// NewContentSelector creates a new ContentSelector
func NewContentSelector(rawDir string, processedMgr *state.ProcessedManager) (*ContentSelector, error) {
	// Regex to match files with line numbers: *-line-<NUM>.md
	regex, err := regexp.Compile(`line-(\d+)\.md$`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile file number regex: %w", err)
	}

	return &ContentSelector{
		rawDir:          rawDir,
		processedMgr:    processedMgr,
		fileNumberRegex: regex,
	}, nil
}

// FileNumber represents a file with its number
type FileNumber struct {
	Path     string
	Filename string
	Number   int
}

// SelectByNumber selects a file by its line number
func (cs *ContentSelector) SelectByNumber(number int) (string, error) {
	files, err := cs.ParseFileNames()
	if err != nil {
		return "", fmt.Errorf("failed to parse file names: %w", err)
	}

	// Find file with matching number
	for _, fn := range files {
		if fn.Number == number {
			// Check if already processed
			if cs.processedMgr.IsProcessed(fn.Filename) {
				return "", fmt.Errorf("file %s (number %d) has already been processed",
					fn.Filename, number)
			}
			return fn.Path, nil
		}
	}

	return "", fmt.Errorf("no file found with number %d", number)
}

// SelectRandom selects a random unprocessed file
func (cs *ContentSelector) SelectRandom() (string, error) {
	files, err := cs.ParseFileNames()
	if err != nil {
		return "", fmt.Errorf("failed to parse file names: %w", err)
	}

	// Filter unprocessed files
	unprocessed := make([]FileNumber, 0)
	processedFiles := cs.processedMgr.GetProcessedFiles()

	for _, fn := range files {
		if !processedFiles[fn.Filename] {
			unprocessed = append(unprocessed, fn)
		}
	}

	if len(unprocessed) == 0 {
		return "", fmt.Errorf("no unprocessed files available")
	}

	// Select random file
	rand.Seed(time.Now().UnixNano())
	selected := unprocessed[rand.Intn(len(unprocessed))]

	return selected.Path, nil
}

// ParseFileNames parses all markdown files in the raw directory
func (cs *ContentSelector) ParseFileNames() ([]FileNumber, error) {
	var files []FileNumber

	err := filepath.WalkDir(cs.rawDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if it's a markdown file
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Check if it has a line number
		filename := filepath.Base(path)
		match := cs.fileNumberRegex.FindStringSubmatch(filename)
		if match != nil && len(match) > 1 {
			var number int
			_, err := fmt.Sscanf(match[1], "%d", &number)
			if err != nil {
				// Invalid number, skip
				return nil
			}

			files = append(files, FileNumber{
				Path:     path,
				Filename: filename,
				Number:   number,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by number
	sort.Slice(files, func(i, j int) bool {
		return files[i].Number < files[j].Number
	})

	return files, nil
}

// GetUnprocessedCount returns the count of unprocessed files
func (cs *ContentSelector) GetUnprocessedCount() (int, error) {
	files, err := cs.ParseFileNames()
	if err != nil {
		return 0, err
	}

	processedFiles := cs.processedMgr.GetProcessedFiles()
	unprocessed := 0

	for _, fn := range files {
		if !processedFiles[fn.Filename] {
			unprocessed++
		}
	}

	return unprocessed, nil
}

// GetTotalCount returns the total count of markdown files
func (cs *ContentSelector) GetTotalCount() (int, error) {
	files, err := cs.ParseFileNames()
	if err != nil {
		return 0, err
	}
	return len(files), nil
}

// GetProgress returns the progress of processing (0.0 to 1.0)
func (cs *ContentSelector) GetProgress() (float64, error) {
	total, err := cs.GetTotalCount()
	if err != nil {
		return 0, err
	}

	if total == 0 {
		return 1.0, nil
	}

	processed := cs.processedMgr.GetCount()
	return float64(processed) / float64(total), nil
}
