package orchestrator

import (
	"fmt"
	"time"

	"github.com/aka/semarang/pkg/agents"
	"github.com/aka/semarang/pkg/config"
	"github.com/aka/semarang/pkg/services"
	"github.com/aka/semarang/pkg/state"
)

// Orchestrator coordinates the video generation pipeline
type Orchestrator struct {
	config         *config.Config
	contentSelector *agents.ContentSelector
	scriptWriter   *agents.ScriptWriter
	videoGenerator *agents.VideoGenerator
	qualityChecker *agents.QualityChecker
	processedMgr   *state.ProcessedManager
	ttsUsageMgr    *state.TTSUsageManager
	contentParser  *services.ContentParser
	ttsService     *services.TTSService
}

// NewOrchestrator creates a new Orchestrator
func NewOrchestrator(cfg *config.Config) (*Orchestrator, error) {
	// Initialize state managers
	processedMgr, err := state.NewProcessedManager(cfg.GetProcessedStateFile())
	if err != nil {
		return nil, fmt.Errorf("failed to create processed manager: %w", err)
	}

	ttsUsageMgr, err := state.NewTTSUsageManager(cfg.GetTTSUsageStateFile(), cfg.TTSMaxCharsPerDay)
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS usage manager: %w", err)
	}

	// Initialize content selector
	contentSelector, err := agents.NewContentSelector(cfg.GetRawDir(), processedMgr)
	if err != nil {
		return nil, fmt.Errorf("failed to create content selector: %w", err)
	}

	// Initialize content parser
	contentParser, err := services.NewContentParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create content parser: %w", err)
	}

	// Initialize LLM service
	llmService := services.NewLLMService(cfg.ZAIAPIKey, cfg.ZAIAPIURL)

	// Initialize TTS service
	ttsService := services.NewTTSService(
		cfg.YandexAPIKey,
		cfg.YandexFolderID,
		cfg.YandexTTSURL,
		cfg.YandexVoice,
		cfg.YandexLanguage,
		ttsUsageMgr,
		cfg.TTSMaxCharsPerDay,
	)

	// Initialize video service
	videoService := services.NewVideoService(
		cfg.PuppeteerServiceURL,
		cfg.VideoWidth,
		cfg.VideoHeight,
	)

	// Initialize script writer
	scriptWriter := agents.NewScriptWriter(
		llmService,
		cfg.TTSSpeechRateCharsPerSec,
		cfg.VideoMaxDurationSeconds*1000,
		cfg.GetOutputScriptsDir(),
	)

	// Initialize video generator
	videoGenerator := agents.NewVideoGenerator(
		ttsService,
		videoService,
		cfg.GetOutputAudioDir(),
		cfg.GetOutputVideosDir(),
	)

	// Initialize quality checker
	qualityChecker := agents.NewQualityChecker(
		cfg.VideoMaxDurationSeconds,
		100, // 100MB max file size
	)

	return &Orchestrator{
		config:         cfg,
		contentSelector: contentSelector,
		scriptWriter:   scriptWriter,
		videoGenerator: videoGenerator,
		qualityChecker: qualityChecker,
		processedMgr:   processedMgr,
		ttsUsageMgr:    ttsUsageMgr,
		contentParser:  contentParser,
		ttsService:     ttsService,
	}, nil
}

// Run runs the pipeline for a specific file number or a random unprocessed file
func (o *Orchestrator) Run(fileNumber *int) error {
	fmt.Println("=" + string(make([]byte, 60)) + "=")
	fmt.Println("  Semarang - YouTube Shorts Production Pipeline")
	fmt.Println("=" + string(make([]byte, 60)) + "=")
	fmt.Println()

	startTime := time.Now()

	// Print TTS quota info
	used, remaining, total := o.ttsService.GetQuotaInfo()
	calls := o.ttsService.GetCallsToday()
	fmt.Printf("TTS Quota: %d/%d used, %d remaining (%d calls today)\n", used, total, remaining, calls)
	fmt.Println()

	// Select content
	var filePath string
	var err error

	if fileNumber != nil {
		fmt.Printf("Selecting file with number: %d\n", *fileNumber)
		filePath, err = o.contentSelector.SelectByNumber(*fileNumber)
	} else {
		fmt.Println("Selecting random unprocessed file...")
		filePath, err = o.contentSelector.SelectRandom()
	}

	if err != nil {
		return fmt.Errorf("failed to select content: %w", err)
	}

	fmt.Printf("Selected: %s\n\n", filePath)

	// Parse content
	fmt.Println("Parsing markdown file...")
	content, err := o.contentParser.Parse(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse content: %w", err)
	}

	fmt.Printf("Title: %s\n", content.Title)
	fmt.Printf("Code blocks: %d, Diagrams: %d\n\n", len(content.CodeBlocks), len(content.Diagrams))

	// Generate script
	fmt.Println("Generating video script...")
	script, err := o.scriptWriter.WriteScript(content)
	if err != nil {
		return fmt.Errorf("failed to generate script: %w", err)
	}

	fmt.Printf("Script generated: %d segments, %dms total\n\n", len(script.Segments), script.TotalDurationMs)

	// Generate video
	fmt.Println("Generating video...")
	videoPath, err := o.videoGenerator.Generate(script)
	if err != nil {
		return fmt.Errorf("failed to generate video: %w", err)
	}

	fmt.Printf("Video generated: %s\n\n", videoPath)

	// Quality check
	fmt.Println("Running quality check...")
	result, err := o.qualityChecker.Check(videoPath)
	if err != nil {
		return fmt.Errorf("quality check failed: %w", err)
	}

	if !result.Passed {
		return fmt.Errorf("quality check failed:\n%s", o.qualityChecker.GetDetailedReport(result))
	}

	// Mark as processed
	filename := filePath[len(o.config.GetRawDir())+1:]
	today := o.config.GetTodayString()
	err = o.processedMgr.MarkProcessed(filename, today, videoPath, "completed")
	if err != nil {
		return fmt.Errorf("failed to mark as processed: %w", err)
	}

	// Print summary
	duration := time.Since(startTime)
	fmt.Println()
	fmt.Println("=" + string(make([]byte, 60)) + "=")
	fmt.Println("  Pipeline completed successfully!")
	fmt.Println("=" + string(make([]byte, 60)) + "=")
	fmt.Printf("  Output: %s\n", videoPath)
	fmt.Printf("  Duration: %v\n", duration.Round(time.Millisecond))
	fmt.Println()

	// Print progress info
	totalCount, _ := o.contentSelector.GetTotalCount()
	progress, _ := o.contentSelector.GetProgress()
	fmt.Printf("Progress: %.1f%% (%d/%d files)\n", progress*100, o.processedMgr.GetCount(), totalCount)

	return nil
}

// RunDaily runs the pipeline for a daily video generation
func (o *Orchestrator) RunDaily() error {
	return o.Run(nil)
}

// GetProgress returns the current progress
func (o *Orchestrator) GetProgress() (float64, int, int) {
	progress, _ := o.contentSelector.GetProgress()
	total, _ := o.contentSelector.GetTotalCount()
	return progress, o.processedMgr.GetCount(), total
}

// GetTTSInfo returns TTS usage information
func (o *Orchestrator) GetTTSInfo() (used, remaining, total int, callsToday int) {
	used, remaining, total = o.ttsService.GetQuotaInfo()
	callsToday = o.ttsService.GetCallsToday()
	return
}
