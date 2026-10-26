package bastion

import (
	"slices"
	"testing"
)

func TestValidName(t *testing.T) {
	for _, name := range []string{"my-bastion", "fumi-shizuka-tsuki", "a1", "db.tool_2"} {
		if !validName.MatchString(name) {
			t.Errorf("%q should be valid", name)
		}
	}
	for _, name := range []string{"", "a", "-bad", ".bad", "has space", "日本語"} {
		if validName.MatchString(name) {
			t.Errorf("%q should be invalid", name)
		}
	}
}

func TestPromptEnv(t *testing.T) {
	if got, want := promptEnv("sh", "fumi-ao-ike"), []string{`PS1=fumi-ao-ike:\w \$ `}; !slices.Equal(got, want) {
		t.Errorf("promptEnv(sh) = %q, want %q", got, want)
	}
	if got := promptEnv("zsh", "fumi-ao-ike"); got != nil {
		t.Errorf("promptEnv(zsh) = %q, want nil (zsh uses a different prompt syntax)", got)
	}
}
