package models

// VideoSpec — спецификация для генерации видео
type VideoSpec struct {
	Slug        string
	AudioPath   string   // путь к WAV файлу
	CodeBlocks  []CodeBlock
	OutputPath  string
	Width       int
	Height      int
	FPS         int
}

// ProductionResult — результат производства одного видео
type ProductionResult struct {
	FileNum    int
	Slug       string
	VideoPath  string
	AudioPath  string
	ScriptPath string
	DurationSec float64
	Success    bool
	Error      string
}
