package devicedetector

import "testing"

func TestDetectBotKnownBots(t *testing.T) {
	tests := []struct {
		name         string
		userAgent    string
		wantName     string
		wantCategory string
	}{
		{"googlebot", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", "Googlebot", "Search bot"},
		{"bingbot", "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)", "Bingbot", "Search bot"},
		{"facebookexternalhit", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)", "Facebook", "Social Media Agent"},
		{"whatsapp", "WhatsApp/2.23.20.0 A", "WhatsApp", "Social Media Agent"},
		{"ahrefsbot", "Mozilla/5.0 (compatible; AhrefsBot/7.0; +http://ahrefs.com/robot/)", "AhrefsBot", "Crawler"},
		{"yandexbot", "Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)", "Yandex Bot", "Search bot"},
		{"pinterestbot", "Pinterestbot/1.0 (+https://www.pinterest.com/bot.html)", "Pinterest", "Social Media Agent"},
		{"pinterest legacy crawler", "Pinterest/0.2 (+https://www.pinterest.com/bot.html)", "Pinterest", "Social Media Agent"},
		{"generic catch-all", "Mozilla/5.0 (compatible; ExampleService crawler/1.0)", "Generic Bot", "Generic"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectBot(tt.userAgent)
			if got == nil {
				t.Fatalf("detectBot(%q) = nil, want %q", tt.userAgent, tt.wantName)
			}
			if got.Name != tt.wantName || got.Category != tt.wantCategory {
				t.Errorf("detectBot(%q) = {%q,%q}, want {%q,%q}", tt.userAgent, got.Name, got.Category, tt.wantName, tt.wantCategory)
			}
		})
	}
}

func TestDetectBotNonBot(t *testing.T) {
	tests := []struct {
		name string
		ua   string
	}{
		// A real mobile device whose UA embeds "bot" inside an app name.
		{"botim app", "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Mobile Safari/537.36 Botim/1.0"},
		// The Pinterest mobile app (not the crawler) carries a bare Pinterest/ token.
		{"pinterest android app", "Mozilla/5.0 (Linux; Android 8.0.0; XT1635-02) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0 Mobile Safari/537.36 [Pinterest/Android]"},
		{"pinterest ios app", "Pinterest/11.30.0 (iPhone; iOS 16.0; Scale/3.00)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectBot(tt.ua); got != nil {
				t.Errorf("detectBot(%q) = %+v, want nil (false positive)", tt.ua, got)
			}
		})
	}
}

func TestLoadBotRules(t *testing.T) {
	if _, err := loadBotRules([]byte(`[{"regex":"googlebot","name":"Googlebot","category":"Search bot"}]`)); err != nil {
		t.Errorf("loadBotRules(valid) error = %v", err)
	}
	if _, err := loadBotRules([]byte(`not json`)); err == nil {
		t.Error("loadBotRules(bad json) error = nil, want error")
	}
	if _, err := loadBotRules([]byte(`[{"regex":"(","name":"x","category":"y"}]`)); err == nil {
		t.Error("loadBotRules(bad regex) error = nil, want error")
	}
	if _, err := loadBotRules([]byte(`[{"regex":"","name":"x","category":"y"}]`)); err == nil {
		t.Error("loadBotRules(empty regex) error = nil, want error")
	}
	if _, err := loadBotRules([]byte(`[{"regex":"x","name":"","category":"y"}]`)); err == nil {
		t.Error("loadBotRules(empty name) error = nil, want error")
	}
}

func TestBotRulesLoaded(t *testing.T) {
	if len(botRules) == 0 {
		t.Fatal("botRules is empty; bots.json failed to load")
	}
}

func TestMustLoadBotRulesPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("mustLoadBotRules(bad data) did not panic")
		}
	}()
	mustLoadBotRules([]byte(`not json`))
}
