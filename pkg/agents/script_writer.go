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
	llm  *services.LLMService
	lang string
}

func NewScriptWriter(llm *services.LLMService, lang string) *ScriptWriter {
	if lang == "" {
		lang = "ru"
	}
	return &ScriptWriter{llm: llm, lang: lang}
}

// llmSegment — один сегмент из ответа LLM
type llmSegment struct {
	Text string `json:"text"`
	SSML string `json:"ssml"`
}

// langNames — название языка для промпта
var langNames = map[string]string{
	"ru": "русский",
	"en": "English",
	"es": "español",
}

// buildSystemPrompt формирует системный промпт с учётом языка вывода
func buildSystemPrompt(lang string) string {
	langName, ok := langNames[lang]
	if !ok {
		lang = "ru"
		langName = "русский"
	}

	narrativeLang := fmt.Sprintf("- Язык нарратива и субтитров: %s (%s)", langName, lang)
	codeLang := fmt.Sprintf("- Комментарии в коде пиши на %s", langName)

	var ssmlSection string
	if lang == "ru" {
		ssmlSection = `Правила SSML (поле ssml каждого сегмента) — синтезатор SaluteSpeech:

Поле ssml — это полноценная озвучка сегмента с богатой SSML-разметкой. Используй все доступные инструменты:

1. Логическое ударение — выдели *звёздочками* самое важное слово в предложении:
   Пример: "это *ключевое* поведение горутин"

2. Паузы — расставляй <break time="300ms"/> там где оратор делал бы вдох или смысловую паузу:
   Пример: "мьютекс блокирует доступ. <break time="300ms"/> Только один поток входит."

3. Go-термины и иностранные слова — ОБЯЗАТЕЛЬНО транслитерируй в русскую фонетику с ударением:
   Пример: "используй goroutine" → "используй горут+ину"
	 Пример: splitHostPort → "сплит-хост-порт"

4. Ударения в остальных словах - НЕ НАДО РАССТАВЛЯТЬ, кроме того
   НЕЛЬЗЯ ставить "+" если это часть кода: Ctrl+C → пиши Ctrl-C

5. Аббревиатуры — читай по буквам:
   <say-as interpret-as="characters">API</say-as> → "эй-пи-ай"
   Применяй к: API, HTTP, HTTPS, SQL, URL, gRPC, UUID, CSV, JWT, TLS и т.п.

6. Имена файлов: output.go → output точка go
   Числовые порты: 8080 → восемь ноль восемь ноль

7. Не используй восклицательный знак. Используй только знак вопроса или точку для окончания предложений.

НЕ оборачивай сегмент в <speak> — добавляется автоматически`
	} else {
		ssmlSection = `SSML rules for each segment (SaluteSpeech synthesizer):

1. Logical emphasis — wrap the key word in *asterisks*: "this is the *key* behaviour"
2. Pauses — use <break time="300ms"/> at natural breath points
3. Abbreviations — spell out with <say-as interpret-as="characters">API</say-as>
   Apply to: API, HTTP, HTTPS, SQL, URL, gRPC, UUID, CSV, JWT, TLS etc.
4. File names: output.go → "output dot go"
   Numeric ports: 8080 → "eight zero eight zero"

Do NOT wrap segment in <speak> — it is added automatically`
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

%s

Code rules:
%s
- If there is Go code → return the best fragment, up to 15 lines, codeLang: "go"
- If no code or trivial → write a bash example (brew install / run command), codeLang: "bash"
- NO emoji in code (❌, ✅, 🔥 etc.) — use text comments: // ERROR, // OK
- Add a comment to the code not on the same line, but before the line being commented.

Reply with ONLY JSON:
{
  "segments": [
    {"text": "clean segment text (for subtitles)", "ssml": "text with SSML markup (for voice)"},
    {"text": "next segment.", "ssml": "next *segment* <break time=\"200ms\"/>."}
  ],
  "title": "short title for file name (latin, hyphens)",
  "code": "code block to display",
  "codeLang": "go"
}`, narrativeLang, ssmlSection, codeLang)
}

// Write генерирует сценарий для видео из контента
func (sw *ScriptWriter) Write(content *models.RawContent) (*models.Script, error) {
	systemPrompt := buildSystemPrompt(sw.lang)

	mainCode := ""
	if len(content.CodeBlocks) > 0 {
		mainCode = content.CodeBlocks[0].Code
	} else if len(content.OldBlocks) > 0 {
		mainCode = content.OldBlocks[0]
	}

	userPrompt := fmt.Sprintf("Секрет Go #%d:\n\n%s\n\nКод:\n```go\n%s\n```",
		content.FileNum, content.Explanation, mainCode)

	raw, err := sw.llm.Complete(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("script_writer LLM: %w", err)
	}

	llmSegs, title, displayCode, displayLang, err := parseScriptJSON(raw)
	if err != nil {
		// Fallback: весь ответ как один сегмент
		text := strings.TrimSpace(raw)
		llmSegs = []llmSegment{{Text: text, SSML: text}}
		title = fmt.Sprintf("go-secret-%d", content.FileNum)
	}

	// Собираем NarrationText (чистый) и NarrationSSML из сегментов
	textParts := make([]string, 0, len(llmSegs))
	ssmlParts := make([]string, 0, len(llmSegs))
	for _, s := range llmSegs {
		textParts = append(textParts, s.Text)
		ssml := s.SSML
		if ssml == "" {
			ssml = s.Text
		}
		ssmlParts = append(ssmlParts, ssml)
	}
	narrationText := strings.Join(textParts, " ")
	narrationSSML := "<speak>" + strings.Join(ssmlParts, " ") + "</speak>"

	// Обрезаем если вышли за лимит (safety net)
	narrationText = truncateToFit(narrationText, maxDurationSec)

	chars := utf8.RuneCountInString(narrationText)
	duration := float64(chars) / russianCharsPerSec

	script := &models.Script{
		FileNum:       content.FileNum,
		Slug:          sanitizeSlug(title),
		SourceFile:    content.FilePath,
		NarrationText: narrationText,
		NarrationSSML: narrationSSML,
		TotalSeconds:  math.Round(duration*10) / 10,
		Segments:      buildSegments(llmSegs),
		DisplayCode:   displayCode,
		DisplayLang:   displayLang,
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

var reNonSlug = regexp.MustCompile(`[^a-z0-9\-]`)

func sanitizeSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = reNonSlug.ReplaceAllString(s, "")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "go-secret"
	}
	return s
}
