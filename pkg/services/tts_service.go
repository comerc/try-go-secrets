package services

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/genai"

	"go-secrets-pipeline/pkg/config"
)

const (
	geminiTTSChannels      = uint16(1)
	geminiTTSSampleRateHz  = uint32(24000)
	geminiTTSBitsPerSample = uint16(16)
)

type TTSService struct {
	cfg *config.Config
}

func NewTTSService(cfg *config.Config) *TTSService {
	return &TTSService{cfg: cfg}
}

func (s *TTSService) SelectVoice(existing string) (string, error) {
	if existing = strings.TrimSpace(existing); existing != "" {
		return existing, nil
	}
	if voice := strings.TrimSpace(s.cfg.GeminiTTSVoice); voice != "" {
		return voice, nil
	}

	voices, err := readVoices("x-voices.md")
	if err != nil {
		return "", err
	}
	if len(voices) == 0 {
		return "", fmt.Errorf("x-voices.md does not contain voices")
	}

	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(voices))))
	if err != nil {
		return "", fmt.Errorf("select random voice: %w", err)
	}
	return voices[idx.Int64()], nil
}

// Synthesize генерирует WAV файл из текста и сохраняет его по outputPath.
func (s *TTSService) Synthesize(text, voice, outputPath string) error {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  s.cfg.GeminiAPIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return fmt.Errorf("Gemini TTS client: %w", err)
	}

	resp, err := client.Models.GenerateContent(ctx, s.cfg.GeminiTTSModel, genai.Text(text), &genai.GenerateContentConfig{
		ResponseModalities: []string{"AUDIO"},
		SpeechConfig: &genai.SpeechConfig{
			VoiceConfig: &genai.VoiceConfig{
				PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
					VoiceName: voice,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("Gemini TTS synthesize: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return fmt.Errorf("Gemini TTS response did not include audio data")
	}

	audio := resp.Candidates[0].Content.Parts[0].InlineData
	if audio == nil || len(audio.Data) == 0 {
		return fmt.Errorf("Gemini TTS response did not include inline audio data")
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	if err := writeWAV(outputPath, audio.Data, geminiTTSChannels, geminiTTSSampleRateHz, geminiTTSBitsPerSample); err != nil {
		return fmt.Errorf("write WAV: %w", err)
	}
	return nil
}

func readVoices(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()

	var voices []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		voice, _, _ := strings.Cut(line, ",")
		voice = strings.TrimSpace(voice)
		if voice != "" {
			voices = append(voices, voice)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return voices, nil
}

func writeWAV(fileName string, pcm []byte, channels uint16, sampleRate uint32, bitsPerSample uint16) error {
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	byteRate := sampleRate * uint32(channels) * uint32(bitsPerSample) / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := uint32(len(pcm))

	if _, err := file.WriteString("RIFF"); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(36)+dataSize); err != nil {
		return err
	}
	if _, err := file.WriteString("WAVEfmt "); err != nil {
		return err
	}
	for _, value := range []any{
		uint32(16),
		uint16(1),
		channels,
		sampleRate,
		byteRate,
		blockAlign,
		bitsPerSample,
	} {
		if err := binary.Write(file, binary.LittleEndian, value); err != nil {
			return err
		}
	}
	if _, err := file.WriteString("data"); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, dataSize); err != nil {
		return err
	}
	_, err = file.Write(pcm)
	return err
}
