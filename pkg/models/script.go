package models

import (
	"strings"
)

// Script represents a video script with segments
type Script struct {
	Title           string    `json:"title"`
	Date            string    `json:"date"` // YYYY-MM-DD
	Segments        []Segment `json:"segments"`
	TotalDurationMs int       `json:"total_duration_ms"`
	SourceFile      string    `json:"source_file"`
}

// Segment represents a segment in the video script
type Segment struct {
	Type        string `json:"type"` // "voice" or "code" or "diagram"
	Text        string `json:"text,omitempty"`
	Content     string `json:"content,omitempty"`
	DurationMs  int    `json:"duration_ms"`
	TTSFile     string `json:"tts_file,omitempty"` // Path to generated audio
}

// NewScript creates a new Script
func NewScript(title, date, sourceFile string) *Script {
	return &Script{
		Title:      title,
		Date:       date,
		Segments:   make([]Segment, 0),
		SourceFile: sourceFile,
	}
}

// AddVoiceSegment adds a voice segment to the script
func (s *Script) AddVoiceSegment(text string, durationMs int) {
	s.Segments = append(s.Segments, Segment{
		Type:       "voice",
		Text:       text,
		DurationMs: durationMs,
	})
}

// AddCodeSegment adds a code segment to the script
func (s *Script) AddCodeSegment(code string, durationMs int) {
	s.Segments = append(s.Segments, Segment{
		Type:       "code",
		Content:    code,
		DurationMs: durationMs,
	})
}

// AddDiagramSegment adds a diagram segment to the script
func (s *Script) AddDiagramSegment(diagram string, durationMs int) {
	s.Segments = append(s.Segments, Segment{
		Type:       "diagram",
		Content:    diagram,
		DurationMs: durationMs,
	})
}

// CalculateTotalDuration calculates the total duration of all segments
func (s *Script) CalculateTotalDuration() int {
	total := 0
	for _, seg := range s.Segments {
		total += seg.DurationMs
	}
	s.TotalDurationMs = total
	return total
}

// GetVoiceSegments returns all voice segments
func (s *Script) GetVoiceSegments() []Segment {
	voice := make([]Segment, 0)
	for _, seg := range s.Segments {
		if seg.Type == "voice" {
			voice = append(voice, seg)
		}
	}
	return voice
}

// GetVisualSegments returns all visual segments (code, diagram)
func (s *Script) GetVisualSegments() []Segment {
	visual := make([]Segment, 0)
	for _, seg := range s.Segments {
		if seg.Type == "code" || seg.Type == "diagram" {
			visual = append(visual, seg)
		}
	}
	return visual
}

// GetVoiceText returns concatenated text from all voice segments
func (s *Script) GetVoiceText() string {
	text := ""
	for _, seg := range s.Segments {
		if seg.Type == "voice" {
			text += seg.Text + " "
		}
	}
	return strings.TrimSpace(text)
}

// TrimToDuration trims segments to fit within a maximum duration
// Returns true if trimming occurred
func (s *Script) TrimToDuration(maxDurationMs int) bool {
	if s.CalculateTotalDuration() <= maxDurationMs {
		return false
	}

	currentDuration := 0
	trimmed := make([]Segment, 0)

	for _, seg := range s.Segments {
		if currentDuration+seg.DurationMs <= maxDurationMs {
			trimmed = append(trimmed, seg)
			currentDuration += seg.DurationMs
		} else {
			// Partial segment
			remaining := maxDurationMs - currentDuration
			if remaining > 0 {
				if seg.Type == "voice" {
					// Trim text proportionally
					ratio := float64(remaining) / float64(seg.DurationMs)
					textLen := int(float64(len(seg.Text)) * ratio)
					trimmed = append(trimmed, Segment{
						Type:       seg.Type,
						Text:       seg.Text[:textLen],
						DurationMs: remaining,
					})
				} else {
					// Can't trim code/diagram, just include it shorter
					trimmed = append(trimmed, Segment{
						Type:       seg.Type,
						Content:    seg.Content,
						DurationMs: remaining,
					})
				}
			}
			break
		}
	}

	s.Segments = trimmed
	s.CalculateTotalDuration()
	return true
}

// GetSlug returns a URL-safe slug from the title
func (s *Script) GetSlug() string {
	slug := s.Title
	slug = strings.ToLower(slug)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.ReplaceAll(slug, "/", "-")
	// Limit slug length
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return slug
}
