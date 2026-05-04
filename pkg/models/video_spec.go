package models

// VideoSpec — спецификация для генерации видео
type VideoSpec struct {
	AudioPath     string // путь к WAV файлу
	CodeBlocks    []CodeBlock
	OutputPath    string
	Width         int
	Height        int
	FPS           int
	TerminalTitle string
}

// ProductionResult — результат производства одного видео
type ProductionResult struct {
	FileNum     int
	VideoPath   string
	AudioPath   string
	ScriptPath  string
	DurationSec float64
	Success     bool
	Error       string
}
