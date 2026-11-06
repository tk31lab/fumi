package dockerutil

import "testing"

func TestValidateName(t *testing.T) {
	for _, name := range []string{"my-bastion", "fumi-shizuka-tsuki", "a1", "db.tool_2"} {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", name, err)
		}
	}
	for _, name := range []string{"", "a", "-bad", ".bad", "has space", "日本語"} {
		if ValidateName(name) == nil {
			t.Errorf("ValidateName(%q) = nil, want an error", name)
		}
	}
}
