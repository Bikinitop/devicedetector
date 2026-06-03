package devicedetector

import "testing"

func TestDetect(t *testing.T) {
	d := New()

	tests := []struct {
		name      string
		userAgent string
		want      DeviceType
	}{
		{
			name:      "iphone is mobile",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15",
			want:      Mobile,
		},
		{
			name:      "ipad is tablet",
			userAgent: "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15",
			want:      Tablet,
		},
		{
			name:      "windows desktop",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			want:      Desktop,
		},
		{
			name:      "googlebot is a bot",
			userAgent: "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			want:      Bot,
		},
		{
			name:      "empty string is unknown",
			userAgent: "",
			want:      Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.Detect(tt.userAgent)
			if got.Type != tt.want {
				t.Errorf("Detect(%q).Type = %v, want %v", tt.userAgent, got.Type, tt.want)
			}
			if got.UserAgent != tt.userAgent {
				t.Errorf("Detect(%q).UserAgent = %q, want it preserved", tt.userAgent, got.UserAgent)
			}
		})
	}
}
