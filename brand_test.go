package devicedetector

import "testing"

func TestDetectBrand(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want string
	}{
		{"iphone", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)", "Apple"},
		{"ipad", "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X)", "Apple"},
		{"mac", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)", "Apple"},
		{"samsung", "Mozilla/5.0 (Linux; Android 13; SM-G991B) AppleWebKit/537.36", "Samsung"},
		{"pixel", "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36", "Google"},
		{"xiaomi", "Mozilla/5.0 (Linux; Android 12; Redmi Note 11) AppleWebKit/537.36", "Xiaomi"},
		{"huawei", "Mozilla/5.0 (Linux; Android 10; HUAWEI VOG-L29) AppleWebKit/537.36", "Huawei"},
		{"oneplus", "Mozilla/5.0 (Linux; Android 13; ONEPLUS A6003) AppleWebKit/537.36", "OnePlus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectBrand(tt.ua); got != tt.want {
				t.Errorf("detectBrand(%q) = %q, want %q", tt.ua, got, tt.want)
			}
		})
	}
}

func TestDetectBrandUnknown(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	if got := detectBrand(ua); got != "" {
		t.Errorf("detectBrand(windows) = %q, want \"\"", got)
	}
}

func TestBrandRulesLoaded(t *testing.T) {
	if len(brandRules) == 0 {
		t.Fatal("brandRules is empty; brands.json failed to load")
	}
}

func TestDetectBrandNexus(t *testing.T) {
	// A real Nexus device brands as Google...
	if got := detectBrand("Mozilla/5.0 (Linux; Android 6.0.1; Nexus 5 Build/MOB30M) AppleWebKit/537.36"); got != "Google" {
		t.Errorf("Nexus 5 brand = %q, want Google", got)
	}
	// ...but "nexus" inside an unrelated product token must not.
	if got := detectBrand("Mozilla/5.0 (Windows NT 10.0; Win64; x64) Connexus/2.0"); got == "Google" {
		t.Errorf("Connexus mis-detected as Google (nexus not anchored)")
	}
}

func TestDetectBrandLegacySamsungPrefix(t *testing.T) {
	// Older Samsung device tokens (GT-/SCH-/SGH-, not SM-) must still brand as
	// Samsung — consistent with the model extractor recognizing them.
	cases := []struct{ name, ua string }{
		{"sch", "Mozilla/5.0 (Linux; U; Android 4.4.2; en-us; SCH-I535 Build/KOT49H) AppleWebKit/534.30"},
		{"gt", "Mozilla/5.0 (Linux; U; Android 4.3; en-us; GT-I9300 Build/JSS15J) AppleWebKit/534.30"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := detectBrand(c.ua); got != "Samsung" {
				t.Errorf("detectBrand(%q) = %q, want Samsung", c.ua, got)
			}
		})
	}
}

func TestDetectBrandFromDeviceNotBrowser(t *testing.T) {
	// Samsung Internet (SamsungBrowser) on a non-Samsung device must not
	// override the actual device brand — brand is device-based, not browser-based.
	ua := "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/21.0 Chrome/115.0.0.0 Mobile Safari/537.36"
	if got := detectBrand(ua); got != "Google" {
		t.Errorf("detectBrand(%q) = %q, want Google (not Samsung from the browser token)", ua, got)
	}
}

func TestDetectBrandNoSubstringFalsePositive(t *testing.T) {
	// Brand tokens must be anchored so they don't match inside unrelated words.
	cases := []struct{ name, ua string }{
		{"lipad not apple", "Mozilla/5.0 (Linux; Android 13; Lipad-X1) AppleWebKit/537.36"},
		{"experia not sony", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ExperiaSoft/3.0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := detectBrand(c.ua); got != "" {
				t.Errorf("detectBrand(%q) = %q, want \"\" (no substring false positive)", c.ua, got)
			}
		})
	}
}
