package namegen

import (
	"regexp"
	"testing"
)

// validName mirrors the container name rule enforced by the Docker daemon.
var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`)

func TestGenerateFormat(t *testing.T) {
	for range 100 {
		name := Generate()
		if !validName.MatchString(name) {
			t.Fatalf("generated name %q is not a valid container name", name)
		}
		if !regexp.MustCompile(`^fumi-[a-z]+-[a-z]+$`).MatchString(name) {
			t.Fatalf("generated name %q does not match fumi-<adjective>-<noun>", name)
		}
	}
}

func TestGenerateDeterministic(t *testing.T) {
	got := generate(func(int) int { return 0 })
	if want := "fumi-shizuka-ishi"; got != want {
		t.Fatalf("generate() = %q, want %q", got, want)
	}
}

func TestWordsAreUnique(t *testing.T) {
	for _, words := range [][]string{adjectives, nouns} {
		seen := map[string]bool{}
		for _, w := range words {
			if seen[w] {
				t.Errorf("duplicate word %q", w)
			}
			seen[w] = true
		}
	}
}
