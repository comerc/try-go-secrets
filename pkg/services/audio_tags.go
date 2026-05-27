package services

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var audioTagPattern = regexp.MustCompile(`\[[^\]\[]+\]`)

var geminiAudioTags = map[string]struct{}{
	"[sigh]":           {},
	"[laughing]":       {},
	"[uhm]":            {},
	"[sarcasm]":        {},
	"[robotic]":        {},
	"[shouting]":       {},
	"[whispering]":     {},
	"[extremely fast]": {},
	"[scared]":         {},
	"[curious]":        {},
	"[bored]":          {},
	"[short pause]":    {},
	"[medium pause]":   {},
	"[long pause]":     {},
}

func ExtractAudioTags(text string) []string {
	matches := audioTagPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	tags := make([]string, 0, len(matches))
	for _, tag := range matches {
		tag = strings.ToLower(strings.Join(strings.Fields(tag), " "))
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

func ValidateGeminiAudioTags(text string) error {
	var unsupported []string
	for _, tag := range ExtractAudioTags(text) {
		if _, ok := geminiAudioTags[tag]; !ok {
			unsupported = append(unsupported, tag)
		}
	}
	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("unsupported Gemini TTS audio tags: %s", strings.Join(unsupported, ", "))
}
