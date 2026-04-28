package models

// Script — сгенерированный сценарий видео
type Script struct {
	FileNum       int
	Slug          string
	SourceFile    string
	NarrationText string    // чистый текст для субтитров и оценки длительности
	NarrationSSML string    // SSML-текст для TTS (<speak>...</speak>)
	Segments      []Segment // тайминговые сегменты
	TotalSeconds  float64
	DisplayCode   string // код для отображения в терминале (Go или bash)
	DisplayLang   string // язык: "go" или "bash"
}

type Segment struct {
	Text         string  // чистый текст для субтитров
	StartSec     float64
	DurationSec  float64
	CodeBlockIdx int // -1 если нет кода для этого сегмента
}
