# Brand + Model Detection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Detect device brand and raw model from a User-Agent, exposed as flat `Device.Brand` / `Device.Model` strings.

**Architecture:** Add `matchFirst` (raw-capture loop) to the shared matcher and make `matchVersioned` a thin normalizing wrapper. `detectBrand` uses a curated `brands.json` ruleset; `detectModel` uses two compiled patterns (Apple device word, Android model token). `Detect` populates `Brand`/`Model` for non-bot UAs.

**Tech Stack:** Go 1.25, stdlib only (`embed`, `regexp`, `strings`, `encoding/json`). Spec: `docs/superpowers/specs/2026-06-04-brand-model-detection-design.md`.

---

## File Structure

- **Modify** `match.go` — add `matchFirst`; refactor `matchVersioned` to wrap it.
- **Create** `brand.go` + `brands.json` — `detectBrand`.
- **Create** `model.go` — `detectModel` (two compiled patterns).
- **Modify** `devicedetector.go` — add `Brand`/`Model` to `Device`; populate in `Detect`.
- **Tests**: additions to `match_test.go`; new `brand_test.go`, `model_test.go`; additions to `devicedetector_test.go`.
- **Modify** `README.md`, `CLAUDE.md`.

---

## Task 1: matchFirst (raw capture) + matchVersioned wrapper

**Files:**
- Modify: `match.go`
- Test: `match_test.go`

- [ ] **Step 1: Append the failing tests to `match_test.go`**

```go
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
```

- [ ] **Step 2: Run to verify it fails (does not compile)**

Run: `go test ./... 2>&1 | tail -5`
Expected: build failure — `undefined: matchFirst`.

- [ ] **Step 3: Edit `match.go` — replace the `matchVersioned` function**

Replace this exact block:

```go
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

with:

```go
// matchFirst returns the name and raw capture group 1 of the first rule that
// matches userAgent (file order, first match wins), or ok=false if none match.
// capture is "" when the regex has no capture group.
func matchFirst(rules []versionedRule, userAgent string) (name, capture string, ok bool) {
	for _, rule := range rules {
		m := rule.re.FindStringSubmatch(userAgent)
		if m == nil {
			continue
		}
		if len(m) > 1 {
			capture = m[1]
		}
		return rule.name, capture, true
	}
	return "", "", false
}

// matchVersioned is matchFirst with the capture treated as a version: "_" is
// normalized to "." (for UA tokens like iOS "17_0"). The version is "" when the
// regex has no capture group or no rule matches.
func matchVersioned(rules []versionedRule, userAgent string) (name, version string, ok bool) {
	name, capture, ok := matchFirst(rules, userAgent)
	return name, strings.ReplaceAll(capture, "_", "."), ok
}
```

(`match.go` still imports `strings` — used by the wrapper.)

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./... 2>&1 | tail -5`
Expected: PASS (new test + all pre-existing OS/browser tests, which still see normalized versions).

- [ ] **Step 5: Commit**

```bash
git add match.go match_test.go
git commit -m "refactor: add matchFirst raw-capture matcher; matchVersioned wraps it"
```

---

## Task 2: Brand detection

**Files:**
- Create: `brands.json`
- Create: `brand.go`
- Test: `brand_test.go`

- [ ] **Step 1: Create `brands.json`**

```json
[
  { "regex": "iphone|ipad|ipod|macintosh", "name": "Apple" },
  { "regex": "(?:^|[^a-z])sm-[a-z0-9]+|samsung", "name": "Samsung" },
  { "regex": "(?:^|[^a-z])pixel|nexus", "name": "Google" },
  { "regex": "redmi|poco|xiaomi", "name": "Xiaomi" },
  { "regex": "huawei|(?:^|[^a-z])honor", "name": "Huawei" },
  { "regex": "oneplus", "name": "OnePlus" },
  { "regex": "(?:^|[^a-z])oppo", "name": "Oppo" },
  { "regex": "(?:^|[^a-z])realme", "name": "Realme" },
  { "regex": "(?:^|[^a-z])(?:moto|xt\\d)", "name": "Motorola" },
  { "regex": "xperia", "name": "Sony" },
  { "regex": "nokia", "name": "Nokia" },
  { "regex": "(?:^|[^a-z])asus", "name": "Asus" }
]
```

- [ ] **Step 2: Write `brand_test.go`**

```go
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
```

- [ ] **Step 3: Run to verify it fails (does not compile)**

Run: `go test ./... 2>&1 | tail -5`
Expected: build failure — `undefined: detectBrand`, `brandRules`.

- [ ] **Step 4: Create `brand.go`**

```go
package devicedetector

import _ "embed"

//go:embed brands.json
var brandData []byte

// brandRules holds the ordered, compiled brand rules from brands.json. Short or
// ambiguous tokens are anchored to avoid substring false positives.
var brandRules = mustLoadVersionedRules(brandData)

// detectBrand returns the device brand for userAgent, or "" if none matches.
func detectBrand(userAgent string) string {
	name, _, _ := matchFirst(brandRules, userAgent)
	return name
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./... 2>&1 | tail -5`
Expected: PASS. If any brand case fails, do NOT change the expected brand — investigate the regex/ordering in `brands.json`; report BLOCKED if it cannot pass with the given data.

- [ ] **Step 6: Commit**

```bash
git add brands.json brand.go brand_test.go
git commit -m "feat: add device brand detection"
```

---

## Task 3: Model detection

**Files:**
- Create: `model.go`
- Test: `model_test.go`

- [ ] **Step 1: Write `model_test.go`**

```go
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
```

- [ ] **Step 2: Run to verify it fails (does not compile)**

Run: `go test ./... 2>&1 | tail -5`
Expected: build failure — `undefined: detectModel`.

- [ ] **Step 3: Create `model.go`**

```go
package devicedetector

import "regexp"

// appleModelRe matches Apple's generic device word. Apple does not expose the
// specific model in the User-Agent.
var appleModelRe = regexp.MustCompile(`(?i)(iphone|ipad|ipod)`)

// androidModelRe captures the raw Android model token between the OS version
// and the next "Build" or ")".
var androidModelRe = regexp.MustCompile(`(?i)android [\w.]+; ?([^;)]+?)(?: build|\))`)

// detectModel returns the raw device model for userAgent, or "" if none
// matches. The model is returned verbatim (e.g. "Pixel 7", "SM-G991B"); it is
// not normalized or mapped to a marketing name. Apple reports only the device
// word (iPhone/iPad/iPod).
func detectModel(userAgent string) string {
	if m := appleModelRe.FindStringSubmatch(userAgent); m != nil {
		return m[1]
	}
	if m := androidModelRe.FindStringSubmatch(userAgent); m != nil {
		return m[1]
	}
	return ""
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./... 2>&1 | tail -5`
Expected: PASS. If a model case fails, do NOT change the expectation — investigate the regex; report BLOCKED if it cannot pass.

- [ ] **Step 5: Commit**

```bash
git add model.go model_test.go
git commit -m "feat: add raw device model extraction"
```

---

## Task 4: Wire brand + model into Detect

**Files:**
- Modify: `devicedetector.go`
- Test: `devicedetector_test.go`

- [ ] **Step 1: Append the failing integration tests to `devicedetector_test.go`**

```go
func TestDetectPopulatesBrandAndModel(t *testing.T) {
	d := New()
	dev := d.Detect("Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36")
	if dev.Brand != "Google" {
		t.Errorf("Brand = %q, want Google", dev.Brand)
	}
	if dev.Model != "Pixel 7" {
		t.Errorf("Model = %q, want Pixel 7", dev.Model)
	}
}

func TestDetectBotHasNoBrandOrModel(t *testing.T) {
	d := New()
	dev := d.Detect("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	if dev.Type != Bot {
		t.Fatalf("Type = %v, want Bot", dev.Type)
	}
	if dev.Brand != "" || dev.Model != "" {
		t.Errorf("bot got Brand=%q Model=%q, want both empty", dev.Brand, dev.Model)
	}
}
```

- [ ] **Step 2: Run to verify it fails (does not compile)**

Run: `go test ./... 2>&1 | tail -5`
Expected: build failure — `dev.Brand undefined` / `dev.Model undefined`.

- [ ] **Step 3: Add `Brand`/`Model` fields to `Device` in `devicedetector.go`**

Replace this exact block:

```go
	// OS and Browser are populated for non-bot User-Agents when a rule
	// matches; they are nil for bots or when nothing matches.
	OS      *OSInfo
	Browser *BrowserInfo
}
```

with:

```go
	// OS and Browser are populated for non-bot User-Agents when a rule
	// matches; they are nil for bots or when nothing matches.
	OS      *OSInfo
	Browser *BrowserInfo
	// Brand and Model are the raw device brand and model for non-bot
	// User-Agents; "" when unknown. Model is verbatim, not mapped to a
	// marketing name.
	Brand string
	Model string
}
```

- [ ] **Step 4: Populate `Brand`/`Model` in `Detect`**

Replace this exact block:

```go
	return Device{
		Type:      classify(strings.ToLower(userAgent)),
		UserAgent: userAgent,
		OS:        detectOS(userAgent),
		Browser:   detectBrowser(userAgent),
	}
}
```

with:

```go
	return Device{
		Type:      classify(strings.ToLower(userAgent)),
		UserAgent: userAgent,
		OS:        detectOS(userAgent),
		Browser:   detectBrowser(userAgent),
		Brand:     detectBrand(userAgent),
		Model:     detectModel(userAgent),
	}
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./... 2>&1 | tail -5`
Expected: PASS (all tests).

- [ ] **Step 6: Commit**

```bash
git add devicedetector.go devicedetector_test.go
git commit -m "feat: populate Device.Brand and Device.Model from Detect"
```

---

## Task 5: Coverage and docs

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Confirm fmt/vet/tests/coverage**

Run:
```bash
gofmt -l . && go vet ./... && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
```
Expected: gofmt empty, vet clean, tests PASS, total coverage `> 90%`. If `<= 90%`, report BLOCKED with `go tool cover -func=coverage.out`. (coverage.out is gitignored.)

- [ ] **Step 2: Document brand/model in `README.md`**

In the `## OS and browser` section, replace this exact paragraph:

```markdown
Detection uses embedded, curated, ordered regex rulesets (`oss.json`,
`browsers.json`); the version is taken verbatim from the User-Agent (e.g.
Windows reports the NT version `10.0`, not the marketing name).
```

with:

```markdown
Detection uses embedded, curated, ordered regex rulesets (`oss.json`,
`browsers.json`); the version is taken verbatim from the User-Agent (e.g.
Windows reports the NT version `10.0`, not the marketing name).

`Detect` also sets `Device.Brand` and `Device.Model` (both `""` when unknown):

```go
device := d.Detect("Mozilla/5.0 (Linux; Android 13; Pixel 7) ... Chrome/120.0.0.0 Mobile Safari/537.36")
fmt.Println(device.Brand, "-", device.Model) // Google - Pixel 7
```

The brand comes from a curated token ruleset (`brands.json`); the model is the
**raw** identifier from the User-Agent (e.g. `SM-G991B`, not "Galaxy S21"), and
Apple reports only the device word (`iPhone`/`iPad`).
```

- [ ] **Step 3: Document brand/model in `CLAUDE.md`**

In the `## Architecture` section, immediately after the `- **OS and browser detection**` bullet, add:

```markdown
- **Brand and model detection** (`brand.go`, `model.go`) run for non-bot UAs:
  `detectBrand` matches a curated `brands.json` token ruleset (via the shared
  `matchFirst`, the raw-capture sibling of `matchVersioned`); `detectModel`
  uses two patterns (Apple device word; the Android model token between the OS
  version and `Build`/`)`). `Detect` sets `Device.Brand`/`Device.Model`
  (`""` for bots / no match). The model is raw — no marketing-name mapping.
```

- [ ] **Step 4: Verify and commit**

Run: `grep -n 'Device.Brand' README.md && grep -n 'Brand and model detection' CLAUDE.md`
Expected: both found.

```bash
git add README.md CLAUDE.md
git commit -m "docs: document brand and model detection"
```

---

## After the plan

Run the PR merge gate (`CLAUDE.md`): push `feat/brand-model-detection`, open a
PR, confirm CI (`test`/`lint`) is green, then `/simplify` → `/code-review` →
address Codex review comments → merge once green. After merge, release-please
will draft a `v0.3.0` Release PR.

## Self-Review notes

- **Spec coverage:** `matchFirst` + `matchVersioned` wrapper (Task 1); brand
  ruleset + `detectBrand` (Task 2); `detectModel` Apple + Android patterns
  (Task 3); `Device.Brand`/`Device.Model` + Detect wiring + bot→empty (Task 4);
  coverage + docs (Task 5). Raw-model / no-mapping, brand anchoring,
  panic-on-bad-data are all covered. Non-goals respected (no mapping, no PC
  vendor, generic Apple model).
- **Type consistency:** `matchFirst`, `matchVersioned`, `detectBrand`,
  `brandRules`/`brandData`, `detectModel`, `appleModelRe`/`androidModelRe`,
  `Device.Brand`/`Device.Model` are used identically across tasks.
- **Placeholders:** none — every file's full content / exact edit is given.
