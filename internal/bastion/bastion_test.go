package bastion

import (
	"slices"
	"testing"
)

func TestPromptEnv(t *testing.T) {
	if got, want := promptEnv("sh", "fumi-ao-ike"), []string{`PS1=fumi-ao-ike:\w \$ `}; !slices.Equal(got, want) {
		t.Errorf("promptEnv(sh) = %q, want %q", got, want)
	}
	if got := promptEnv("zsh", "fumi-ao-ike"); got != nil {
		t.Errorf("promptEnv(zsh) = %q, want nil (zsh uses a different prompt syntax)", got)
	}
}
