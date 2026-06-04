package devicedetector

import "testing"

func TestDetectModel(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want string
	}{
		{"iphone", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)", "iPhone"},
		{"ipad", "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X)", "iPad"},
		{"pixel", "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36", "Pixel 7"},
		{"samsung build", "Mozilla/5.0 (Linux; Android 13; SM-G991B Build/TP1A.220624.014) AppleWebKit/537.36", "SM-G991B"},
		{"redmi", "Mozilla/5.0 (Linux; Android 12; Redmi Note 11) AppleWebKit/537.36", "Redmi Note 11"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectModel(tt.ua); got != tt.want {
				t.Errorf("detectModel(%q) = %q, want %q", tt.ua, got, tt.want)
			}
		})
	}
}

func TestDetectModelNone(t *testing.T) {
	// Windows desktop and Mac carry no model token.
	for _, ua := range []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15",
	} {
		if got := detectModel(ua); got != "" {
			t.Errorf("detectModel(%q) = %q, want \"\"", ua, got)
		}
	}
}
