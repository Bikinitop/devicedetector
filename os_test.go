package devicedetector

import "testing"

func TestDetectOS(t *testing.T) {
	tests := []struct {
		name        string
		userAgent   string
		wantName    string
		wantVersion string
	}{
		{"iphone", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15", "iOS", "17.0"},
		{"ipad", "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15", "iOS", "17.0"},
		{"macos", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15", "macOS", "10.15.7"},
		{"android", "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36", "Android", "13"},
		{"windows", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36", "Windows", "10.0"},
		{"chromeos", "Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36", "Chrome OS", "14541.0.0"},
		{"linux", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36", "Linux", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectOS(tt.userAgent)
			if got == nil {
				t.Fatalf("detectOS(%q) = nil, want %q", tt.userAgent, tt.wantName)
			}
			if got.Name != tt.wantName || got.Version != tt.wantVersion {
				t.Errorf("detectOS(%q) = {%q,%q}, want {%q,%q}", tt.userAgent, got.Name, got.Version, tt.wantName, tt.wantVersion)
			}
		})
	}
}

func TestDetectOSNoMatch(t *testing.T) {
	if got := detectOS("some opaque token with no OS"); got != nil {
		t.Errorf("detectOS(no match) = %+v, want nil", got)
	}
}

func TestOSRulesLoaded(t *testing.T) {
	if len(osRules) == 0 {
		t.Fatal("osRules is empty; oss.json failed to load")
	}
}

func TestDetectOSAndroidWithoutVersion(t *testing.T) {
	// An Android UA that omits the version must still report Android (with an
	// empty version), not fall through to Linux.
	ua := "Mozilla/5.0 (Linux; U; Android; en-us) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1"
	got := detectOS(ua)
	if got == nil || got.Name != "Android" || got.Version != "" {
		t.Errorf("detectOS(%q) = %+v, want {Android, \"\"}", ua, got)
	}
}
