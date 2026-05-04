package agents

import (
	"testing"

	"go-secrets-pipeline/pkg/models"
)

func TestNormalizeSubtitleTextFrenchKeepsPunctuationAttached(t *testing.T) {
	got := normalizeSubtitleText("sous charge ? Le secret: oui !", "fr")
	want := "sous charge\u202f? Le secret: oui\u202f!"
	if got != want {
		t.Fatalf("normalizeSubtitleText() = %q, want %q", got, want)
	}
}

func TestNormalizeSubtitleTextNonFrenchDoesNotChangePunctuation(t *testing.T) {
	got := normalizeSubtitleText("under load ?", "en-us")
	want := "under load ?"
	if got != want {
		t.Fatalf("normalizeSubtitleText() = %q, want %q", got, want)
	}
}

func TestBuildSubtitleWordsFrenchKeepsDetachedQuestionWithWord(t *testing.T) {
	words := buildSubtitleWords([]models.Segment{
		{
			Text:        "sous charge ?",
			DurationSec: 3,
		},
	}, 1, "fr")

	if len(words) != 2 {
		t.Fatalf("len(words) = %d, want 2: %+v", len(words), words)
	}
	if words[1].Word != "charge\u202f?" {
		t.Fatalf("words[1].Word = %q, want charge\\u202f?", words[1].Word)
	}
}
