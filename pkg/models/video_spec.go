package models

// VideoSpec represents a specification for video generation
type VideoSpec struct {
	// Output
	OutputPath string `json:"output_path"`

	// Dimensions
	Width  int `json:"width"`
	Height int `json:"height"`

	// Content
	Code    string `json:"code,omitempty"`
	Diagram string `json:"diagram,omitempty"`

	// Animation
	DurationMs      int    `json:"duration_ms"`
	TypingSpeedMs   int    `json:"typing_speed_ms"` // Delay between characters
	FontSize        int    `json:"font_size"`
	Theme          string `json:"theme"` // "dark" or "light"

	// Audio
	AudioPath string `json:"audio_path,omitempty"`
}

// NewVideoSpec creates a new VideoSpec for code animation
func NewVideoSpec(outputPath string, width, height int, code string, durationMs int) *VideoSpec {
	return &VideoSpec{
		OutputPath:    outputPath,
		Width:         width,
		Height:        height,
		Code:          code,
		DurationMs:    durationMs,
		TypingSpeedMs: 50,  // 50ms per character
		FontSize:      16,
		Theme:         "dark",
	}
}

// NewVideoSpecWithAudio creates a new VideoSpec with audio
func NewVideoSpecWithAudio(outputPath string, width, height int, code string, durationMs int, audioPath string) *VideoSpec {
	spec := NewVideoSpec(outputPath, width, height, code, durationMs)
	spec.AudioPath = audioPath
	return spec
}

// NewVideoSpecForDiagram creates a new VideoSpec for diagram animation
func NewVideoSpecForDiagram(outputPath string, width, height int, diagram string, durationMs int) *VideoSpec {
	return &VideoSpec{
		OutputPath: outputPath,
		Width:      width,
		Height:     height,
		Diagram:    diagram,
		DurationMs:  durationMs,
		FontSize:   16,
		Theme:      "dark",
	}
}

// ToMap converts VideoSpec to a map for JSON serialization
func (vs *VideoSpec) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"output_path":      vs.OutputPath,
		"width":           vs.Width,
		"height":          vs.Height,
		"code":            vs.Code,
		"diagram":         vs.Diagram,
		"duration_ms":     vs.DurationMs,
		"typing_speed_ms": vs.TypingSpeedMs,
		"font_size":       vs.FontSize,
		"theme":           vs.Theme,
		"audio_path":      vs.AudioPath,
	}
}
