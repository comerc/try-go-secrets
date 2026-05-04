package models

// Script — сгенерированный сценарий видео
type Script struct {
	FileNum       int
	Title         string
	Voice         string
	SourceFile    string
	NarrationText string    // чистый текст для субтитров и оценки длительности
	NarrationTags string    // текст с Audio-Tags в квадратных скобках для TTS
	Segments      []Segment // тайминговые сегменты
	TotalSeconds  float64
	Code          string // код для отображения в терминале (Go или bash)
	CodeLang      string // язык: "go" или "bash"
}

type Segment struct {
	Text         string // чистый текст для субтитров
	StartSec     float64
	DurationSec  float64
	CodeBlockIdx int // -1 если нет кода для этого сегмента
}
