package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aka/semarang/pkg/models"
	"github.com/aka/semarang/pkg/services"
)

// ScriptWriter handles creating video scripts from parsed content
type ScriptWriter struct {
	llmService           *services.LLMService
	speechRateCharsPerSec float64
	maxDurationMs        int
	scriptsDir           string
}

// NewScriptWriter creates a new ScriptWriter
func NewScriptWriter(
	llmService *services.LLMService,
	speechRateCharsPerSec float64,
	maxDurationMs int,
	scriptsDir string,
) *ScriptWriter {
	return &ScriptWriter{
		llmService:           llmService,
		speechRateCharsPerSec: speechRateCharsPerSec,
		maxDurationMs:        maxDurationMs,
		scriptsDir:           scriptsDir,
	}
}

// WriteScript creates a video script from parsed content
func (sw *ScriptWriter) WriteScript(content *models.ParsedContent) (*models.Script, error) {
	today := time.Now().Format("2006-01-02")

	// Generate script text via LLM
	var scriptText string
	var err error

	if len(content.CodeBlocks) > 0 {
		// Generate script with code
		scriptText, err = sw.llmService.GenerateScript(
			content.Title,
			content.Explanation,
			content.CodeBlocks[0].Content,
		)
	} else {
		// Generate script without code
		scriptText, err = sw.llmService.GenerateScriptSimplified(
			content.Title,
			content.Explanation,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate script: %w", err)
	}

	// Create script object
	script := models.NewScript(content.Title, today, content.FilePath)

	// Add voice segment
	voiceDurationMs := int(float64(len(scriptText)) / sw.speechRateCharsPerSec * 1000)
	script.AddVoiceSegment(scriptText, voiceDurationMs)

	// Add code segment if available
	if len(content.CodeBlocks) > 0 {
		// Estimate code animation duration (typically 10-15 seconds)
		codeDurationMs := 12000
		script.AddCodeSegment(content.CodeBlocks[0].Content, codeDurationMs)
	}

	// Add diagram segment if available
	if len(content.Diagrams) > 0 {
		// Estimate diagram animation duration
		diagramDurationMs := 8000
		script.AddDiagramSegment(content.Diagrams[0].Content, diagramDurationMs)
	}

	// Calculate total duration
	script.CalculateTotalDuration()

	// Trim to max duration if needed
	if script.TotalDurationMs > sw.maxDurationMs {
		sw.logInfo("Script duration %dms exceeds max %dms, trimming...",
			script.TotalDurationMs, sw.maxDurationMs)
		script.TrimToDuration(sw.maxDurationMs)
	}

	// Save script to file
	if err := sw.saveScript(script); err != nil {
		return nil, fmt.Errorf("failed to save script: %w", err)
	}

	return script, nil
}

// saveScript saves the script to a JSON file
func (sw *ScriptWriter) saveScript(script *models.Script) error {
	// Ensure scripts directory exists
	if err := os.MkdirAll(sw.scriptsDir, 0755); err != nil {
		return err
	}

	// Generate filename
	filename := fmt.Sprintf("%s.json", script.Date)
	filepath := filepath.Join(sw.scriptsDir, filename)

	// Marshal script to JSON
	data, err := json.MarshalIndent(script, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(filepath, data, 0644)
}

// LoadScript loads a script from a JSON file
func (sw *ScriptWriter) LoadScript(date string) (*models.Script, error) {
	filename := fmt.Sprintf("%s.json", date)
	filepath := filepath.Join(sw.scriptsDir, filename)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var script models.Script
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, err
	}

	return &script, nil
}

// logInfo logs an info message
func (sw *ScriptWriter) logInfo(format string, args ...interface{}) {
	fmt.Printf("[ScriptWriter] "+format+"\n", args...)
}
