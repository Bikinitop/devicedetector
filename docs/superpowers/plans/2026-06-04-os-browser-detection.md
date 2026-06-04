# OS + Browser Detection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Detect operating-system and browser name + version from a User-Agent, exposed as flat `Device.OS` and `Device.Browser` fields, reusing the bot engine's data-driven pattern with added version extraction.

**Architecture:** A shared versioned-rule matcher (`match.go`) loads embedded JSON rulesets and extracts a name + version (regex capture group 1, `_`→`.` normalized). `os.go`/`browser.go` are thin detectors over `oss.json`/`browsers.json`. `Detect` populates `OS`/`Browser` for non-bot UAs. The shared `compileInsensitive` helper is also retrofitted into `bot.go`.

**Tech Stack:** Go 1.25, stdlib only (`embed`, `encoding/json`, `regexp`, `strings`). Spec: `docs/superpowers/specs/2026-06-04-os-browser-detection-design.md`.

---

## File Structure

- **Create** `match.go` — `compileInsensitive`, `versionedRule`, `versionedRuleJSON`, `loadVersionedRules`/`mustLoadVersionedRules`, `matchVersioned`.
- **Modify** `bot.go` — use `compileInsensitive` in `loadBotRules` (remove duplicated `(?i)`/compile/empty-regex logic).
- **Create** `os.go` + `oss.json` — `OSInfo`, `detectOS`.
- **Create** `browser.go` + `browsers.json` — `BrowserInfo`, `detectBrowser`.
- **Modify** `devicedetector.go` — add `OS`/`Browser` to `Device`; populate in `Detect`.
- **Tests**: `match_test.go`, `os_test.go`, `browser_test.go`, and additions to `devicedetector_test.go`.
- **Modify** `README.md`, `CLAUDE.md`.

---

## Task 1: Shared versioned-rule matcher + bot.go refactor

**Files:**
- Create: `match.go`
- Modify: `bot.go`
- Test: `match_test.go`

- [ ] **Step 1: Write the failing tests in `match_test.go`**

```go
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

	// Rule with no capture group yields an empty version.
	name, version, ok = matchVersioned(rules, "X11; Linux x86_64")
	if !ok || name != "Linux" || version != "" {
		t.Errorf("matchVersioned linux = (%q,%q,%v), want (Linux,\"\",true)", name, version, ok)
	}

	// Underscores in the captured version are normalized to dots.
	uRules := mustLoadVersionedRules([]byte(`[{"regex":"iphone os (\\d+[_\\d]+)","name":"iOS"}]`))
	name, version, ok = matchVersioned(uRules, "CPU iPhone OS 17_0 like Mac OS X")
	if !ok || name != "iOS" || version != "17.0" {
		t.Errorf("matchVersioned ios = (%q,%q,%v), want (iOS,17.0,true)", name, version, ok)
	}

	if _, _, ok := matchVersioned(rules, "totally unrelated"); ok {
		t.Error("matchVersioned(no match) ok = true, want false")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail (do not compile)**

Run: `go test ./... 2>&1 | tail -5`
Expected: build failure — `undefined: compileInsensitive`, `loadVersionedRules`, `mustLoadVersionedRules`, `matchVersioned`.

- [ ] **Step 3: Create `match.go`**

```go
package devicedetector

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// compileInsensitive validates a non-empty pattern and compiles it
// case-insensitively. Centralizing the "(?i)" flag means individual rules in
// the embedded JSON files cannot silently forget it.
func compileInsensitive(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, fmt.Errorf("empty regex")
	}
	return regexp.Compile("(?i)" + pattern)
}

// versionedRule is a compiled rule that yields a name and a version. The
// version is capture group 1 of the regex (empty when the regex has no group).
type versionedRule struct {
	re   *regexp.Regexp
	name string
}

// versionedRuleJSON is the on-disk shape of a rule in oss.json / browsers.json.
type versionedRuleJSON struct {
	Regex string `json:"regex"`
	Name  string `json:"name"`
}

// loadVersionedRules parses and compiles a versioned ruleset.
func loadVersionedRules(data []byte) ([]versionedRule, error) {
	var raw []versionedRuleJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse rules: %w", err)
	}
	rules := make([]versionedRule, 0, len(raw))
	for i, r := range raw {
		if r.Name == "" {
			return nil, fmt.Errorf("rule at index %d has empty name", i)
		}
		re, err := compileInsensitive(r.Regex)
		if err != nil {
			return nil, fmt.Errorf("bad rule at index %d (%q): %w", i, r.Regex, err)
		}
		rules = append(rules, versionedRule{re: re, name: r.Name})
	}
	return rules, nil
}

// mustLoadVersionedRules is loadVersionedRules for author-controlled embedded
// data: a parse/compile failure is a programmer error, so it panics.
func mustLoadVersionedRules(data []byte) []versionedRule {
	rules, err := loadVersionedRules(data)
	if err != nil {
		panic("devicedetector: " + err.Error())
	}
	return rules
}

// matchVersioned returns the name and version of the first rule that matches
// userAgent (file order, first match wins), or ok=false if none match. version
// is capture group 1 with "_" normalized to "." (for UA tokens like "17_0"),
// or "" when the regex has no capture group.
func matchVersioned(rules []versionedRule, userAgent string) (name, version string, ok bool) {
	for _, rule := range rules {
		m := rule.re.FindStringSubmatch(userAgent)
		if m == nil {
			continue
		}
		if len(m) > 1 {
			version = strings.ReplaceAll(m[1], "_", ".")
		}
		return rule.name, version, true
	}
	return "", "", false
}
```

- [ ] **Step 4: Refactor `bot.go` to use `compileInsensitive`**

In `bot.go`, replace the loop body inside `loadBotRules`:

```go
	for i, r := range raw {
		if r.Regex == "" || r.Name == "" {
			return nil, fmt.Errorf("bot rule at index %d has empty regex or name", i)
		}
		// Case-insensitivity is enforced centrally so each rule in bots.json
		// need not carry (and cannot silently forget) a "(?i)" flag.
		re, err := regexp.Compile("(?i)" + r.Regex)
		if err != nil {
			return nil, fmt.Errorf("bad bot regex at index %d (%q): %w", i, r.Regex, err)
		}
		rules = append(rules, botRule{re: re, name: r.Name, category: r.Category})
	}
```

with:

```go
	for i, r := range raw {
		if r.Name == "" {
			return nil, fmt.Errorf("bot rule at index %d has empty name", i)
		}
		re, err := compileInsensitive(r.Regex)
		if err != nil {
			return nil, fmt.Errorf("bad bot rule at index %d (%q): %w", i, r.Regex, err)
		}
		rules = append(rules, botRule{re: re, name: r.Name, category: r.Category})
	}
```

(`bot.go` keeps importing `regexp` — `botRule.re` is `*regexp.Regexp`. `compileInsensitive` now rejects the empty regex, so the existing `loadBotRules(empty regex)` test still gets an error.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./... 2>&1 | tail -5`
Expected: PASS (new match tests + all pre-existing bot/device tests).

- [ ] **Step 6: Commit**

```bash
git add match.go match_test.go bot.go
git commit -m "feat: add shared versioned-rule matcher; reuse it in bot loader"
```

---

## Task 2: OS detection

**Files:**
- Create: `oss.json`
- Create: `os.go`
- Test: `os_test.go`

- [ ] **Step 1: Create `oss.json`**

```json
[
  { "regex": "iphone os (\\d+[_\\d]+)", "name": "iOS" },
  { "regex": "cpu os (\\d+[_\\d]+) like mac os x", "name": "iOS" },
  { "regex": "mac os x (\\d+[_\\d]+)", "name": "macOS" },
  { "regex": "android (\\d+[.\\d]*)", "name": "Android" },
  { "regex": "cros \\S+ (\\d+[.\\d]*)", "name": "Chrome OS" },
  { "regex": "windows nt (\\d+[.\\d]*)", "name": "Windows" },
  { "regex": "linux", "name": "Linux" }
]
```

- [ ] **Step 2: Write the failing tests in `os_test.go`**

```go
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
```

- [ ] **Step 3: Run tests to verify they fail (do not compile)**

Run: `go test ./... 2>&1 | tail -5`
Expected: build failure — `undefined: detectOS`, `osRules` (and `OSInfo`).

- [ ] **Step 4: Create `os.go`**

```go
package devicedetector

import _ "embed"

//go:embed oss.json
var osData []byte

// OSInfo describes the operating system parsed from a User-Agent string.
type OSInfo struct {
	Name    string
	Version string
}

// osRules holds the ordered, compiled OS rules from oss.json. Order is
// significant (e.g. iOS before macOS, since an iOS UA contains "Mac OS X").
var osRules = mustLoadVersionedRules(osData)

// detectOS returns the OS for userAgent, or nil if no rule matches.
func detectOS(userAgent string) *OSInfo {
	if name, version, ok := matchVersioned(osRules, userAgent); ok {
		return &OSInfo{Name: name, Version: version}
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./... 2>&1 | tail -5`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add oss.json os.go os_test.go
git commit -m "feat: add OS detection (name + version)"
```

---

## Task 3: Browser detection

**Files:**
- Create: `browsers.json`
- Create: `browser.go`
- Test: `browser_test.go`

- [ ] **Step 1: Create `browsers.json`**

Order is correctness-critical: Chrome-derivatives (Edge/Opera/Samsung/CriOS) and Firefox-on-iOS (FxiOS) come before Chrome/Firefox; real Safari (`Version/<v> … Safari`) comes after the Chrome family.

```json
[
  { "regex": "edg(?:e|ios|a)?/(\\d+[.\\d]*)", "name": "Edge" },
  { "regex": "opr/(\\d+[.\\d]*)", "name": "Opera" },
  { "regex": "opera/(\\d+[.\\d]*)", "name": "Opera" },
  { "regex": "samsungbrowser/(\\d+[.\\d]*)", "name": "Samsung Internet" },
  { "regex": "crios/(\\d+[.\\d]*)", "name": "Chrome" },
  { "regex": "fxios/(\\d+[.\\d]*)", "name": "Firefox" },
  { "regex": "firefox/(\\d+[.\\d]*)", "name": "Firefox" },
  { "regex": "chrome/(\\d+[.\\d]*)", "name": "Chrome" },
  { "regex": "version/(\\d+[.\\d]*).*safari", "name": "Safari" }
]
```

- [ ] **Step 2: Write the failing tests in `browser_test.go`**

```go
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
```

- [ ] **Step 3: Run tests to verify they fail (do not compile)**

Run: `go test ./... 2>&1 | tail -5`
Expected: build failure — `undefined: detectBrowser`, `browserRules` (and `BrowserInfo`).

- [ ] **Step 4: Create `browser.go`**

```go
package devicedetector

import _ "embed"

//go:embed browsers.json
var browserData []byte

// BrowserInfo describes the browser parsed from a User-Agent string.
type BrowserInfo struct {
	Name    string
	Version string
}

// browserRules holds the ordered, compiled browser rules from browsers.json.
// Order is significant: Chrome-derivatives (Edge/Opera/Samsung/CriOS) precede
// Chrome, and real Safari is matched after the Chrome family, because these
// User-Agents nest each other's tokens.
var browserRules = mustLoadVersionedRules(browserData)

// detectBrowser returns the browser for userAgent, or nil if no rule matches.
func detectBrowser(userAgent string) *BrowserInfo {
	if name, version, ok := matchVersioned(browserRules, userAgent); ok {
		return &BrowserInfo{Name: name, Version: version}
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./... 2>&1 | tail -5`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add browsers.json browser.go browser_test.go
git commit -m "feat: add browser detection (name + version)"
```

---

## Task 4: Wire OS + browser into Detect

**Files:**
- Modify: `devicedetector.go`
- Test: `devicedetector_test.go`

- [ ] **Step 1: Append the failing integration tests to `devicedetector_test.go`**

```go
func TestDetectPopulatesOSAndBrowser(t *testing.T) {
	d := New()
	dev := d.Detect("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if dev.OS == nil || dev.OS.Name != "Windows" {
		t.Errorf("OS = %+v, want Windows", dev.OS)
	}
	if dev.Browser == nil || dev.Browser.Name != "Chrome" || dev.Browser.Version != "120.0.0.0" {
		t.Errorf("Browser = %+v, want {Chrome, 120.0.0.0}", dev.Browser)
	}
}

func TestDetectBotHasNoOSOrBrowser(t *testing.T) {
	d := New()
	dev := d.Detect("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	if dev.Type != Bot {
		t.Fatalf("Type = %v, want Bot", dev.Type)
	}
	if dev.OS != nil || dev.Browser != nil {
		t.Errorf("bot got OS=%+v Browser=%+v, want both nil", dev.OS, dev.Browser)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail (do not compile)**

Run: `go test ./... 2>&1 | tail -5`
Expected: build failure — `dev.OS undefined` / `dev.Browser undefined`.

- [ ] **Step 3: Add `OS` and `Browser` fields to `Device` in `devicedetector.go`**

Replace:

```go
// Device holds the facts we extract from a single User-Agent string.
type Device struct {
	Type      DeviceType
	UserAgent string
	// Bot is non-nil only when Type == Bot; it carries the identified bot's
	// name and category.
	Bot *BotInfo
}
```

with:

```go
// Device holds the facts we extract from a single User-Agent string.
type Device struct {
	Type      DeviceType
	UserAgent string
	// Bot is non-nil only when Type == Bot; it carries the identified bot's
	// name and category.
	Bot *BotInfo
	// OS and Browser are populated for non-bot User-Agents when a rule
	// matches; they are nil for bots or when nothing matches.
	OS      *OSInfo
	Browser *BrowserInfo
}
```

- [ ] **Step 4: Populate OS/Browser in `Detect`**

Replace:

```go
func (d *Detector) Detect(userAgent string) Device {
	if bot := detectBot(userAgent); bot != nil {
		return Device{Type: Bot, UserAgent: userAgent, Bot: bot}
	}
	return Device{
		Type:      classify(strings.ToLower(userAgent)),
		UserAgent: userAgent,
	}
}
```

with:

```go
func (d *Detector) Detect(userAgent string) Device {
	if bot := detectBot(userAgent); bot != nil {
		return Device{Type: Bot, UserAgent: userAgent, Bot: bot}
	}
	return Device{
		Type:      classify(strings.ToLower(userAgent)),
		UserAgent: userAgent,
		OS:        detectOS(userAgent),
		Browser:   detectBrowser(userAgent),
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./... 2>&1 | tail -5`
Expected: PASS (all tests).

- [ ] **Step 6: Commit**

```bash
git add devicedetector.go devicedetector_test.go
git commit -m "feat: populate Device.OS and Device.Browser from Detect"
```

---

## Task 5: Coverage check and docs

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Confirm formatting, vet, tests, coverage**

Run:
```bash
gofmt -l . && go vet ./... && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
```
Expected: `gofmt` prints nothing, vet clean, tests PASS, total coverage `> 90%`. If ≤ 90%, report BLOCKED with `go tool cover -func=coverage.out`.

- [ ] **Step 2: Document OS/browser in `README.md`**

Replace the `## Bot detection` heading line:

```markdown
## Bot detection
```

with:

```markdown
## OS and browser

For non-bot User-Agents, `Detect` populates `Device.OS` and `Device.Browser`
(each `nil` when nothing matches):

```go
device := d.Detect("Mozilla/5.0 (Windows NT 10.0; Win64; x64) ... Chrome/120.0.0.0 Safari/537.36")
fmt.Println(device.OS.Name, device.OS.Version)           // Windows 10.0
fmt.Println(device.Browser.Name, device.Browser.Version) // Chrome 120.0.0.0
```

Detection uses embedded, curated, ordered regex rulesets (`oss.json`,
`browsers.json`); the version is taken verbatim from the User-Agent (e.g.
Windows reports the NT version `10.0`, not the marketing name).

## Bot detection
```

- [ ] **Step 3: Document the new dimensions in `CLAUDE.md`**

In the `## Architecture` section, immediately after the bullet that begins
`- **Bot detection runs first**`, add:

```markdown
- **OS and browser detection** (`os.go`, `browser.go`) run for non-bot UAs via
  a shared versioned matcher (`match.go`): embedded ordered regex rulesets
  (`oss.json`, `browsers.json`) whose capture group 1 is the version
  (`_`→`.` normalized). `Detect` sets `Device.OS`/`Device.Browser` (nil for
  bots / no match). Ordering is correctness-critical (iOS before macOS; browser
  derivatives before Chrome/Firefox; Safari after the Chrome family). To add a
  rule, add a `{regex, name}` entry with a version capture group; bad
  data/regex panics at init and a test compiles every rule.
```

- [ ] **Step 4: Verify docs and commit**

Run: `grep -n '## OS and browser' README.md && grep -n 'OS and browser detection' CLAUDE.md`
Expected: both found.

```bash
git add README.md CLAUDE.md
git commit -m "docs: document OS and browser detection"
```

---

## After the plan

Run the PR merge gate (`CLAUDE.md`): push `feat/os-browser-detection`, open a PR,
confirm CI (`test`/`lint`) is green on the PR, then `/simplify` → `/code-review`
→ address Codex review comments → merge once green.

## Self-Review notes

- **Spec coverage:** shared matcher + `compileInsensitive` + bot refactor (Task 1);
  OS dataset/detector (Task 2); browser dataset/detector with ordering (Task 3);
  `Device.OS`/`Device.Browser` + Detect wiring + bot→nil (Task 4); version
  normalization `_`→`.` (Task 1 `matchVersioned`, tested); panic-on-bad-data +
  empty-name/regex rejection (Task 1 tests); coverage + docs (Task 5). All spec
  sections map to a task. Non-goals (no engine/platform/brand/model, no
  NT-mapping, bots get no OS/browser) are respected.
- **Type consistency:** `versionedRule{re,name}`, `versionedRuleJSON{Regex,Name}`,
  `OSInfo{Name,Version}`, `BrowserInfo{Name,Version}`, `osRules`/`browserRules`,
  `detectOS`/`detectBrowser`, `matchVersioned`, `compileInsensitive`,
  `mustLoadVersionedRules` are used identically across tasks.
- **Placeholders:** none — every file's full content / exact edit is given.
