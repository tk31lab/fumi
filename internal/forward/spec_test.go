package forward

import "testing"

func TestParseSpec(t *testing.T) {
	tests := []struct {
		in   string
		want Spec
	}{
		{"8090:web:80", Spec{LocalPort: 8090, Host: "web", RemotePort: 80}},
		{"db:5432", Spec{LocalPort: 5432, Host: "db", RemotePort: 5432}},
		{"15432:myapp-db-1:5432", Spec{LocalPort: 15432, Host: "myapp-db-1", RemotePort: 5432}},
		{"65535:example.com:443", Spec{LocalPort: 65535, Host: "example.com", RemotePort: 443}},
	}
	for _, tt := range tests {
		got, err := ParseSpec(tt.in)
		if err != nil {
			t.Errorf("ParseSpec(%q) error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseSpec(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestParseSpecErrors(t *testing.T) {
	for _, in := range []string{
		"", "web", "80", "a:b:c:d",
		"web:0", "web:65536", "web:http", "web:-1",
		"0:web:80", "x:web:80",
		":80", "8090::80", "my host:80", "a/b:80",
	} {
		if got, err := ParseSpec(in); err == nil {
			t.Errorf("ParseSpec(%q) = %+v, want an error", in, got)
		}
	}
}

func TestSpecString(t *testing.T) {
	s := Spec{LocalPort: 8090, Host: "web", RemotePort: 80}
	if got, want := s.String(), "127.0.0.1:8090 -> web:80"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
