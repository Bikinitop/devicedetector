package devicedetector

import "testing"

func TestCompileInsensitive(t *testing.T) {
	if _, err := compileInsensitive(""); err == nil {
		t.Error("compileInsensitive(\"\") error = nil, want error")
	}
	re, err := compileInsensitive("chrome/(\\d+)")
	if err != nil {
		t.Fatalf("compileInsensitive valid error = %v", err)
	}
	if !re.MatchString("CHROME/120") {
		t.Error("compiled regex is not case-insensitive")
	}
}

func TestLoadVersionedRules(t *testing.T) {
	if _, err := loadVersionedRules([]byte(`[{"regex":"firefox/(\\d+)","name":"Firefox"}]`)); err != nil {
		t.Errorf("loadVersionedRules(valid) error = %v", err)
	}
	if _, err := loadVersionedRules([]byte(`nope`)); err == nil {
		t.Error("loadVersionedRules(bad json) error = nil, want error")
	}
	if _, err := loadVersionedRules([]byte(`[{"regex":"x","name":""}]`)); err == nil {
		t.Error("loadVersionedRules(empty name) error = nil, want error")
	}
	if _, err := loadVersionedRules([]byte(`[{"regex":"(","name":"X"}]`)); err == nil {
		t.Error("loadVersionedRules(bad regex) error = nil, want error")
	}
}

func TestMustLoadVersionedRulesPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("mustLoadVersionedRules(bad data) did not panic")
		}
	}()
	mustLoadVersionedRules([]byte(`nope`))
}

func TestMatchVersioned(t *testing.T) {
	rules := mustLoadVersionedRules([]byte(`[
		{"regex":"crios/(\\d+[.\\d]*)","name":"Chrome"},
		{"regex":"firefox/(\\d+[.\\d]*)","name":"Firefox"},
		{"regex":"linux","name":"Linux"}
	]`))

	name, version, ok := matchVersioned(rules, "Mozilla/5.0 Firefox/121.0")
	if !ok || name != "Firefox" || version != "121.0" {
		t.Errorf("matchVersioned firefox = (%q,%q,%v), want (Firefox,121.0,true)", name, version, ok)
	}

	name, version, ok = matchVersioned(rules, "X11; Linux x86_64")
	if !ok || name != "Linux" || version != "" {
		t.Errorf("matchVersioned linux = (%q,%q,%v), want (Linux,\"\",true)", name, version, ok)
	}

	uRules := mustLoadVersionedRules([]byte(`[{"regex":"iphone os (\\d+[_\\d]+)","name":"iOS"}]`))
	name, version, ok = matchVersioned(uRules, "CPU iPhone OS 17_0 like Mac OS X")
	if !ok || name != "iOS" || version != "17.0" {
		t.Errorf("matchVersioned ios = (%q,%q,%v), want (iOS,17.0,true)", name, version, ok)
	}

	if _, _, ok := matchVersioned(rules, "totally unrelated"); ok {
		t.Error("matchVersioned(no match) ok = true, want false")
	}
}

func TestMatchFirstRawCapture(t *testing.T) {
	rules := mustLoadVersionedRules([]byte(`[{"regex":"sm-([a-z0-9_]+)","name":"Samsung"}]`))
	name, capture, ok := matchFirst(rules, "Mozilla/5.0 (Linux; Android 13; SM-G99_1B)")
	if !ok || name != "Samsung" || capture != "G99_1B" {
		t.Errorf("matchFirst = (%q,%q,%v), want (Samsung, G99_1B, true) — capture must NOT be normalized", name, capture, ok)
	}
	if _, _, ok := matchFirst(rules, "no match here"); ok {
		t.Error("matchFirst(no match) ok = true, want false")
	}
	// matchVersioned still normalizes underscores to dots.
	if _, v, _ := matchVersioned(rules, "Mozilla/5.0 (Linux; Android 13; SM-G99_1B)"); v != "G99.1B" {
		t.Errorf("matchVersioned capture = %q, want G99.1B (normalized)", v)
	}
}
