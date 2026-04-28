package services

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"go-secrets-pipeline/pkg/config"
)

// ttsPhonetics — English Go-terms → Russian phonetics with stress marks.
// Longer/plural forms come first to avoid partial replacements.
var ttsPhonetics = func() []struct {
	re *regexp.Regexp
	to string
} {
	terms := [][2]string{
		{"goroutines", "горут+ины"},
		{"goroutine", "горут+ина"},
		{"mutexes", "мь+ютексы"},
		{"mutex", "мь+ютекс"},
		{"interfaces", "интерф+ейсы"},
		{"interface", "интерф+ейс"},
		{"channels", "к+эналы"},
		{"channel", "к+энал"},
		{"slices", "сл+айсы"},
		{"slice", "сл+айс"},
		{"structs", "стр+укты"},
		{"struct", "стр+укт"},
		{"pointers", "п+оинтеры"},
		{"pointer", "п+оинтер"},
		{"deadlock", "д+едлок"},
		{"benchmark", "б+енчмарк"},
		{"runtime", "р+антайм"},
		{"timeout", "т+аймаут"},
		{"deploy", "депл+ой"},
		{"defer", "диф+ёр"},
		{"panic", "п+эник"},
		{"nil", "н+ил"},
		{"vegeta", "вег+ета"},
		{"wrk", "ворк"},
		{"pprof", "пи-проф"},
		{"slog", "эс-лог"},
		{"grpc", "джи-эр-пи-си"},
		{"https", "эйч-ти-ти-пи-эс"},
		{"http", "эйч-ти-ти-пи"},
		{"json", "джей-сон"},
		{"sql", "эс-кю-эл"},
		{"api", "эй-пи-ай"},
		{"url", "ю-эр-эл"},
		{"golang", "гол+анг"},
	}
	var out []struct {
		re *regexp.Regexp
		to string
	}
	for _, t := range terms {
		out = append(out, struct {
			re *regexp.Regexp
			to string
		}{regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(t[0]) + `\b`), t[1]})
	}
	return out
}()

const (
	saluteSpeechOAuthURL = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	saluteSpeechSynthURL = "https://smartspeech.sber.ru/rest/v1/text:synthesize"
)

type TTSService struct {
	cfg    *config.Config
	mu     sync.Mutex
	token  string
	expiry time.Time
	client *http.Client
}

type oauthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"` // milliseconds Unix
}

func NewTTSService(cfg *config.Config) *TTSService {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // Sber uses Russian CA not in standard pool
	}
	return &TTSService{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second, Transport: transport},
	}
}

// langToBCP47 maps our short lang codes to BCP-47 tags supported by SaluteSpeech.
// Supported: ru-RU, en-US, es-ES. Unsupported (cn, hi, ja) fall back to ru-RU.
func langToBCP47(lang string) string {
	switch lang {
	case "en":
		return "en-US"
	case "es":
		return "es-ES"
	default:
		return "ru-RU"
	}
}

// toSSML оборачивает текст в SSML <speak>.
// Если текст уже содержит <speak>...</speak> (сгенерирован LLM) — очищает и возвращает.
// Иначе применяет фонетические замены (только для ru) и оборачивает с тегом <lang>.
func (s *TTSService) toSSML(text string) string {
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "<speak>") && strings.HasSuffix(trimmed, "</speak>") {
		return trimmed
	}

	lang := s.cfg.Lang
	if lang == "" {
		lang = "ru"
	}

	// Фонетические замены только для русского (fallback path)
	if lang == "ru" {
		for _, p := range ttsPhonetics {
			text = p.re.ReplaceAllString(text, p.to)
		}
	}

	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")

	if lang != "ru" {
		bcp47 := langToBCP47(lang)
		return fmt.Sprintf(`<speak><lang xml:lang="%s">%s</lang></speak>`, bcp47, text)
	}
	return "<speak>" + text + "</speak>"
}

// Synthesize генерирует WAV файл из текста и сохраняет его по outputPath
func (s *TTSService) Synthesize(text, outputPath string) error {
	token, err := s.getToken()
	if err != nil {
		return fmt.Errorf("SaluteSpeech auth: %w", err)
	}

	// Собираем URL с параметрами
	params := url.Values{}
	params.Set("voice", s.cfg.SaluteSpeechVoice)
	params.Set("format", "wav16")
	params.Set("sample_rate_hertz", "24000")
	synthURL := saluteSpeechSynthURL + "?" + params.Encode()

	ssml := s.toSSML(text)
	req, err := http.NewRequest("POST", synthURL, strings.NewReader(ssml))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/ssml")
	req.Header.Set("X-Operation-ID", uuid.New().String())
	req.Header.Set("X-Request-ID", uuid.New().String())

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("SaluteSpeech синтез: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SaluteSpeech ошибка %d: %s", resp.StatusCode, string(body))
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// getToken возвращает валидный OAuth2 токен (с кэшем на 29 минут)
func (s *TTSService) getToken() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != "" && time.Now().Before(s.expiry) {
		return s.token, nil
	}

	creds := base64.StdEncoding.EncodeToString(
		[]byte(s.cfg.SaluteSpeechClientID + ":" + s.cfg.SaluteSpeechClientSecret),
	)

	form := url.Values{}
	form.Set("scope", s.cfg.SaluteSpeechScope)

	req, err := http.NewRequest("POST", saluteSpeechOAuthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+creds)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("RqUID", uuid.New().String())

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("OAuth2 запрос: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OAuth2 ошибка %d: %s", resp.StatusCode, string(body))
	}

	var oauthResp oauthResponse
	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		return "", fmt.Errorf("парсинг OAuth2 ответа: %w", err)
	}

	s.token = oauthResp.AccessToken
	// expires_at в миллисекундах, кэшируем на 29 минут
	if oauthResp.ExpiresAt > 0 {
		s.expiry = time.UnixMilli(oauthResp.ExpiresAt).Add(-1 * time.Minute)
	} else {
		s.expiry = time.Now().Add(29 * time.Minute)
	}

	return s.token, nil
}
