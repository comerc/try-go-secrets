package services

import (
	"reflect"
	"testing"
)

func TestExtractAudioTagsDedupesAndNormalizes(t *testing.T) {
	got := ExtractAudioTags("[short   pause] hello [whispering] there [short pause]")
	want := []string{"[short pause]", "[whispering]"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractAudioTags() = %#v, want %#v", got, want)
	}
}

func TestValidateGeminiAudioTagsAcceptsDocumentedTags(t *testing.T) {
	text := "[sigh] [laughing] [uhm] [sarcasm] [robotic] [shouting] [whispering] [extremely fast] [scared] [curious] [bored] [short pause] [medium pause] [long pause]"
	if err := ValidateGeminiAudioTags(text); err != nil {
		t.Fatalf("ValidateGeminiAudioTags() unexpected error: %v", err)
	}
}

func TestValidateGeminiAudioTagsRejectsLegacyTags(t *testing.T) {
	text := "[thinking] hello [pause: short] [whispers] secret"
	if err := ValidateGeminiAudioTags(text); err == nil {
		t.Fatal("ValidateGeminiAudioTags() expected error")
	}
}
