package devicedetector

import "testing"

func TestDetectBrowser(t *testing.T) {
	tests := []struct {
		name        string
		userAgent   string
		wantName    string
		wantVersion string
	}{
		{"chrome windows", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", "Chrome", "120.0.0.0"},
		{"safari macos", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15", "Safari", "17.0"},
		{"safari ios", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1", "Safari", "17.0"},
		{"chrome ios", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0.0.0 Mobile/15E148 Safari/604.1", "Chrome", "120.0.0.0"},
		{"edge", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0", "Edge", "120.0.0.0"},
		{"firefox", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0", "Firefox", "121.0"},
		{"samsung", "Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/23.0 Chrome/115.0.0.0 Mobile Safari/537.36", "Samsung Internet", "23.0"},
		{"opera", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36 OPR/106.0.0.0", "Opera", "106.0.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectBrowser(tt.userAgent)
			if got == nil {
				t.Fatalf("detectBrowser(%q) = nil, want %q", tt.userAgent, tt.wantName)
			}
			if got.Name != tt.wantName || got.Version != tt.wantVersion {
				t.Errorf("detectBrowser(%q) = {%q,%q}, want {%q,%q}", tt.userAgent, got.Name, got.Version, tt.wantName, tt.wantVersion)
			}
		})
	}
}

func TestDetectBrowserNoMatch(t *testing.T) {
	if got := detectBrowser("opaque token with no browser"); got != nil {
		t.Errorf("detectBrowser(no match) = %+v, want nil", got)
	}
}

func TestBrowserRulesLoaded(t *testing.T) {
	if len(browserRules) == 0 {
		t.Fatal("browserRules is empty; browsers.json failed to load")
	}
}

func TestDetectBrowserEdgeBoundary(t *testing.T) {
	// "edg" embedded inside an ordinary token ("Knowledge/1.0") must NOT be
	// detected as Edge.
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 SomeKnowledge/1.0"
	if got := detectBrowser(ua); got != nil && got.Name == "Edge" {
		t.Errorf("detectBrowser(%q) = %+v, want not Edge", ua, got)
	}
}

func TestDetectBrowserAndroidNotSafari(t *testing.T) {
	// An old Android stock-browser UA ("Version/X Mobile Safari", no Chrome
	// token) must not be mislabeled as Safari — Safari is gated to Apple UAs.
	ua := "Mozilla/5.0 (Linux; U; Android; en-us) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1"
	if got := detectBrowser(ua); got != nil && got.Name == "Safari" {
		t.Errorf("detectBrowser(%q) = %+v, want not Safari", ua, got)
	}
}

func TestDetectBrowserOperaIOS(t *testing.T) {
	// Opera on iOS uses the OPT/ token (not OPR/); it must be Opera, not the
	// Safari fallback that the Version/.*Safari rule would otherwise match.
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1 OPT/4.0.0"
	got := detectBrowser(ua)
	if got == nil || got.Name != "Opera" || got.Version != "4.0.0" {
		t.Errorf("detectBrowser(%q) = %+v, want {Opera, 4.0.0}", ua, got)
	}
}

func TestDetectBrowserVersionTrailingSeparator(t *testing.T) {
	// A trailing dot in the UA version token must not leak into the version.
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0. Safari/537.36"
	got := detectBrowser(ua)
	if got == nil || got.Name != "Chrome" || got.Version != "120.0" {
		t.Errorf("detectBrowser(%q) = %+v, want {Chrome, 120.0}", ua, got)
	}
}
