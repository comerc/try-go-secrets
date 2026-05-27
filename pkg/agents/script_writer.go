package agents

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"go-secrets-pipeline/pkg/models"
	"go-secrets-pipeline/pkg/services"
)

const (
	// Средняя скорость русской речи: ~8-9 символов в секунду
	russianCharsPerSec = 8.5
	maxDurationSec     = 55.0 // целевая длина, чтобы уложиться в 60с
)

type ScriptWriter struct {
	llm *services.LLMService
	tts *services.TTSService

	lang string
}

func NewScriptWriter(llm *services.LLMService, tts *services.TTSService, lang string) *ScriptWriter {
	return &ScriptWriter{llm: llm, tts: tts, lang: lang}
}

// llmSegment — один сегмент из ответа LLM
type llmSegment struct {
	Text string `json:"text"`
	Tags string `json:"tags"`
}

// buildSystemPrompt формирует системный промпт с учётом языка вывода
func buildSystemPrompt(lang string) (string, error) {

	narrativeLang := fmt.Sprintf("- Language/locale for title, narration, and subtitles: %s", lang)
	commentLang := fmt.Sprintf("- Write code comments in the same language/locale: %s", lang)

	narrationSection, err := readAudioTagsInstruction()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`You are creating a script for a YouTube Shorts video about Go secrets.

Narrative rules:
- Length: STRICTLY no more than 400 characters total (clean text across all segments, ~47 seconds)
%s
- Structure: hook → secret insight → example → conclusion
- Go terms in the text field stay in English (struct, goroutine, interface, mutex, etc.) — this is subtitle text
- Do NOT say "in this video", "today we will look at" etc.
- FORBIDDEN: read code aloud — do not say variable names, operators, or Go syntax. Explain the MEANING and BEHAVIOUR.
- FORBIDDEN: use exclamation mark "!" — it creates an unnatural accent in TTS.
- Use only question marks and periods at the end of sentences.
- Audio-Tags contract is strict: in the "tags" field, bracketed tags may only come from the allowed list below.
- If you need emotion, emphasis, or speed that is not in the allowed tag list, express it as natural language in the "tags" text, not as a bracketed tag.

---

%s

---

Code rules:
%s
- If there is Go code → return the best fragment, up to 15 lines, codeLang: "go"
- If no code or trivial → write a bash example (brew install / run command), codeLang: "bash"
- NO emoji in code (❌, ✅, 🔥 etc.) — use text comments: // ERROR, // OK
- Add a comment to the code not on the same line, but before the line being commented.

Reply with ONLY JSON:
{
  "segments": [
    {"text": "clean segment text (for subtitles)", "tags": "text with Audio-Tags markup (for voice)"},
    {"text": "next segment.", "tags": "[pause: short] [emphasized] next segment."}
  ],
	"title": "short title without num of secret",
  "code": "code block to display",
  "codeLang": "go"
}`, narrativeLang, narrationSection, commentLang), nil
}

// Write генерирует сценарий для видео из контента
func (sw *ScriptWriter) Write(content *models.RawContent) (*models.Script, error) {
	systemPrompt, err := buildSystemPrompt(sw.lang)
	if err != nil {
		return nil, fmt.Errorf("build system prompt: %w", err)
	}

	mainCode := ""
	if len(content.CodeBlocks) > 0 {
		mainCode = content.CodeBlocks[0].Code
	} else if len(content.OldBlocks) > 0 {
		mainCode = content.OldBlocks[0]
	}

	userPrompt := fmt.Sprintf("Golang Secret #%d:\n\n%s\n\nCode:\n```go\n%s\n```",
		content.FileNum, content.Explanation, mainCode)

	raw, err := sw.llm.Complete(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("script_writer LLM: %w", err)
	}

	llmSegs, title, code, codeLang, err := parseScriptJSON(raw)
	if err != nil {
		return nil, err
	}

	// Собираем NarrationText (чистый) и NarrationTags из сегментов
	textParts := make([]string, 0, len(llmSegs))
	tagParts := make([]string, 0, len(llmSegs))
	for _, s := range llmSegs {
		textParts = append(textParts, s.Text)
		tags := s.Tags
		if tags == "" {
			tags = s.Text
		}
		tagParts = append(tagParts, tags)
	}
	narrationText := strings.Join(textParts, " ")
	narrationTags := strings.Join(tagParts, " ")
	if err := services.ValidateGeminiAudioTags(narrationTags); err != nil {
		return nil, fmt.Errorf("валидация Audio-Tags: %w", err)
	}

	// Обрезаем если вышли за лимит (safety net)
	narrationText = truncateToFit(narrationText, maxDurationSec)

	chars := utf8.RuneCountInString(narrationText)
	duration := float64(chars) / russianCharsPerSec

	voice, err := sw.tts.SelectVoice("")
	if err != nil {
		return nil, fmt.Errorf("выбор голоса TTS: %w", err)
	}

	script := &models.Script{
		FileNum:       content.FileNum,
		Title:         title,
		Voice:         voice,
		SourceFile:    content.FilePath,
		NarrationText: narrationText,
		NarrationTags: narrationTags,
		TotalSeconds:  math.Round(duration*10) / 10,
		Segments:      buildSegments(llmSegs),
		Code:          code,
		CodeLang:      codeLang,
	}

	return script, nil
}

// Save сохраняет сценарий в JSON файл
func (sw *ScriptWriter) Save(script *models.Script, outputDir string) (string, error) {
	dir := filepath.Join(outputDir, "scripts")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s__%03d.json", date, script.FileNum)
	path := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(script, "", "  ")
	if err != nil {
		return "", err
	}

	return path, os.WriteFile(path, data, 0644)
}

func parseScriptJSON(raw string) (segments []llmSegment, title, code, codeLang string, err error) {
	// Вырезаем JSON из возможной обёртки в markdown
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")
	if m := re.FindStringSubmatch(raw); m != nil {
		raw = m[1]
	}

	var result struct {
		Segments []llmSegment `json:"segments"`
		Title    string       `json:"title"`
		Code     string       `json:"code"`
		CodeLang string       `json:"codeLang"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &result); err != nil {
		return nil, "", "", "", err
	}
	if result.CodeLang == "" {
		result.CodeLang = "go"
	}
	return result.Segments, result.Title, result.Code, result.CodeLang, nil
}

func readAudioTagsInstruction() (string, error) {
	data, err := os.ReadFile("x-audio-tags.md")
	if err != nil {
		return "", fmt.Errorf("read x-audio-tags.md: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func truncateToFit(text string, maxSec float64) string {
	maxChars := int(maxSec * russianCharsPerSec)
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}
	// Обрезаем по последнему предложению
	truncated := string(runes[:maxChars])
	if idx := strings.LastIndexAny(truncated, ".?"); idx > 0 {
		return truncated[:idx+1]
	}
	return truncated
}

func buildSegments(llmSegs []llmSegment) []models.Segment {
	segments := make([]models.Segment, 0, len(llmSegs))
	cursor := 0.0

	for i, seg := range llmSegs {
		chars := utf8.RuneCountInString(seg.Text)
		dur := float64(chars) / russianCharsPerSec

		codeIdx := -1
		if i == 1 { // показываем код во втором сегменте
			codeIdx = 0
		}

		segments = append(segments, models.Segment{
			Text:         seg.Text,
			StartSec:     math.Round(cursor*100) / 100,
			DurationSec:  math.Round(dur*100) / 100,
			CodeBlockIdx: codeIdx,
		})
		cursor += dur
	}
	return segments
}
