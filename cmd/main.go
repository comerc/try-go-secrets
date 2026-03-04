package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/aka/semarang/pkg/config"
	"github.com/aka/semarang/pkg/orchestrator"
)

const version = "1.0.0"

func main() {
	// Set up panic recovery
	defer handlePanic()

	// Parse command line flags
	numberFlag := flag.Int("number", 0, "File number to process (e.g., 43 for *-line-043.md)")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	helpFlag := flag.Bool("help", false, "Print help and exit")
	flag.Parse()

	// Print version
	if *versionFlag {
		fmt.Printf("Semarang v%s\n", version)
		fmt.Println("YouTube Shorts Production Pipeline for Go Secrets")
		os.Exit(0)
	}

	// Print help
	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up logging
	logFile := setupLogging(cfg)
	defer logFile.Close()

	log.Printf("Semarang v%s starting...", version)
	log.Printf("Configuration loaded")
	log.Printf("  Raw directory: %s", cfg.RawDir)
	log.Printf("  Output directory: %s", cfg.OutputDir)
	log.Printf("  State directory: %s", cfg.StateDir)
	log.Printf("  Puppeteer service: %s", cfg.PuppeteerServiceURL)

	// Create orchestrator
	orc, err := orchestrator.NewOrchestrator(cfg)
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Determine which file to process
	var fileNumber *int
	if *numberFlag > 0 {
		fileNumber = numberFlag
		log.Printf("Processing file number: %d", *numberFlag)
	} else {
		log.Println("Processing random unprocessed file")
	}

	// Run the pipeline
	startTime := time.Now()
	err = orc.Run(fileNumber)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("Pipeline failed after %v: %v", duration.Round(time.Millisecond), err)
		os.Exit(1)
	}

	log.Printf("Pipeline completed successfully in %v", duration.Round(time.Millisecond))
}

func handlePanic() {
	if r := recover(); r != nil {
		log.Printf("PANIC: %v", r)
		log.Printf("Stack trace:\n%s", debug.Stack())
		os.Exit(1)
	}
}

func setupLogging(cfg *config.Config) *os.File {
	logDir := filepath.Join(cfg.OutputDir, "logs")
	logFileName := fmt.Sprintf("semarang-%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(logDir, logFileName)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		return nil
	}

	// Set up multi-writer (console + file)
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags)

	log.Printf("Logging to: %s", logPath)
	return logFile
}

func printHelp() {
	fmt.Println("Semarang - YouTube Shorts Production Pipeline for Go Secrets")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  semarang [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -number int")
	fmt.Println("        File number to process (e.g., 43 for *-line-043.md)")
	fmt.Println("  -version")
	fmt.Println("        Print version and exit")
	fmt.Println("  -help")
	fmt.Println("        Print this help and exit")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  semarang")
	fmt.Println("        Generate video for a random unprocessed file")
	fmt.Println()
	fmt.Println("  semarang -number 43")
	fmt.Println("        Generate video for Go_Secret__line-043.md")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  Make sure .env file exists with ZAI_API_KEY and YANDEX_API_KEY")
	fmt.Println()
	fmt.Println("Files are selected from raw/ directory by pattern: *-line-<NUM>.md")
}
