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
			name:      "android phone is mobile (carries the Mobile token)",
			userAgent: "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Mobile Safari/537.36",
			want:      Mobile,
		},
		{
			name:      "android tablet is tablet (omits the Mobile token)",
			userAgent: "Mozilla/5.0 (Linux; Android 10; SM-T870) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36",
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

func TestDeviceTypeString(t *testing.T) {
	tests := []struct {
		t    DeviceType
		want string
	}{
		{Desktop, "desktop"},
		{Mobile, "mobile"},
		{Tablet, "tablet"},
		{Bot, "bot"},
		{Unknown, "unknown"},
		{DeviceType(99), "unknown"}, // out-of-range falls through to the default
	}

	for _, tt := range tests {
		if got := tt.t.String(); got != tt.want {
			t.Errorf("DeviceType(%d).String() = %q, want %q", tt.t, got, tt.want)
		}
	}
}
