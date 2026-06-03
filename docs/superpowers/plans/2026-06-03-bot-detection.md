# Bot Detection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace substring bot matching with a data-driven, ordered-regex ruleset that identifies well-known bots (including tokenless crawlers) and reports each bot's name + category.

**Architecture:** An embedded `bots.json` (our own curated data) is parsed once at package init into a slice of compiled `{regex, name, category}` rules. `Detect` runs bot detection first (case-insensitive RE2 regexes on the raw UA, first-match-wins); a hit returns `Type: Bot` with a populated `*BotInfo`, otherwise it falls through to the existing `classify` for device type. The old `bot`/`spider`/`crawl` substring case is removed from `classify`.

**Tech Stack:** Go 1.25, stdlib only (`embed`, `encoding/json`, `regexp`). No external dependencies. Spec: `docs/superpowers/specs/2026-06-03-bot-detection-design.md`.

---

## File Structure

- **Create** `bots.json` — curated bot ruleset data (array of `{regex, name, category}`).
- **Create** `bot.go` — `BotInfo`, `botRule`, JSON loader (`loadBotRules`/`mustLoadBotRules`), embedded `botRules`, and `detectBot`.
- **Create** `bot_test.go` — unit tests for the loader and `detectBot`.
- **Modify** `devicedetector.go` — add `Bot *BotInfo` field to `Device`; call `detectBot` first in `Detect`; remove the bot case from `classify`.
- **Modify** `devicedetector_test.go` — integration tests asserting `Detect` populates `Bot` and that non-bots get `Bot == nil`.

---

## Task 1: Add the curated bot ruleset data

**Files:**
- Create: `bots.json`

- [ ] **Step 1: Create `bots.json`**

```json
[
  { "regex": "(?i)googlebot", "name": "Googlebot", "category": "Search bot" },
  { "regex": "(?i)google-inspectiontool", "name": "Google InspectionTool", "category": "Search bot" },
  { "regex": "(?i)mediapartners-google", "name": "Google AdSense", "category": "Crawler" },
  { "regex": "(?i)adsbot-google", "name": "Google AdsBot", "category": "Crawler" },
  { "regex": "(?i)google-extended", "name": "Google-Extended", "category": "Crawler" },
  { "regex": "(?i)bingbot", "name": "Bingbot", "category": "Search bot" },
  { "regex": "(?i)bingpreview", "name": "Bing Preview", "category": "Search bot" },
  { "regex": "(?i)slurp", "name": "Yahoo! Slurp", "category": "Search bot" },
  { "regex": "(?i)duckduckbot", "name": "DuckDuckBot", "category": "Search bot" },
  { "regex": "(?i)duckduckgo", "name": "DuckDuckGo", "category": "Search bot" },
  { "regex": "(?i)baiduspider", "name": "Baidu Spider", "category": "Search bot" },
  { "regex": "(?i)yandex(?:bot|images|video|mobilebot)?", "name": "Yandex Bot", "category": "Search bot" },
  { "regex": "(?i)sogou", "name": "Sogou Spider", "category": "Search bot" },
  { "regex": "(?i)exabot", "name": "ExaBot", "category": "Search bot" },
  { "regex": "(?i)applebot", "name": "Applebot", "category": "Search bot" },
  { "regex": "(?i)petalbot", "name": "PetalBot", "category": "Search bot" },
  { "regex": "(?i)facebookexternalhit", "name": "Facebook", "category": "Social Media Agent" },
  { "regex": "(?i)facebookcatalog", "name": "Facebook", "category": "Social Media Agent" },
  { "regex": "(?i)facebot", "name": "Facebook", "category": "Social Media Agent" },
  { "regex": "(?i)meta-externalagent", "name": "Meta External Agent", "category": "Crawler" },
  { "regex": "(?i)twitterbot", "name": "Twitterbot", "category": "Social Media Agent" },
  { "regex": "(?i)linkedinbot", "name": "LinkedInBot", "category": "Social Media Agent" },
  { "regex": "(?i)pinterest(?:bot|/)", "name": "Pinterest", "category": "Social Media Agent" },
  { "regex": "(?i)whatsapp", "name": "WhatsApp", "category": "Social Media Agent" },
  { "regex": "(?i)telegrambot", "name": "TelegramBot", "category": "Social Media Agent" },
  { "regex": "(?i)slackbot", "name": "Slackbot", "category": "Social Media Agent" },
  { "regex": "(?i)slack-imgproxy", "name": "Slack", "category": "Social Media Agent" },
  { "regex": "(?i)discordbot", "name": "Discordbot", "category": "Social Media Agent" },
  { "regex": "(?i)redditbot", "name": "Redditbot", "category": "Social Media Agent" },
  { "regex": "(?i)skypeuripreview", "name": "Skype URI Preview", "category": "Social Media Agent" },
  { "regex": "(?i)gptbot", "name": "GPTBot", "category": "Crawler" },
  { "regex": "(?i)oai-searchbot", "name": "OAI-SearchBot", "category": "Search bot" },
  { "regex": "(?i)chatgpt-user", "name": "ChatGPT-User", "category": "Crawler" },
  { "regex": "(?i)claudebot", "name": "ClaudeBot", "category": "Crawler" },
  { "regex": "(?i)anthropic-ai", "name": "Anthropic AI", "category": "Crawler" },
  { "regex": "(?i)perplexitybot", "name": "PerplexityBot", "category": "Crawler" },
  { "regex": "(?i)ccbot", "name": "CCBot", "category": "Crawler" },
  { "regex": "(?i)bytespider", "name": "Bytespider", "category": "Crawler" },
  { "regex": "(?i)ahrefsbot", "name": "AhrefsBot", "category": "Crawler" },
  { "regex": "(?i)semrushbot", "name": "SemrushBot", "category": "Crawler" },
  { "regex": "(?i)mj12bot", "name": "Majestic-12", "category": "Crawler" },
  { "regex": "(?i)dotbot", "name": "DotBot", "category": "Crawler" },
  { "regex": "(?i)dataforseobot", "name": "DataForSeoBot", "category": "Crawler" },
  { "regex": "(?i)screaming\\s?frog", "name": "Screaming Frog SEO Spider", "category": "Crawler" },
  { "regex": "(?i)pingdom", "name": "Pingdom", "category": "Site Monitor" },
  { "regex": "(?i)uptimerobot", "name": "UptimeRobot", "category": "Site Monitor" },
  { "regex": "(?i)statuscake", "name": "StatusCake", "category": "Site Monitor" },
  { "regex": "(?i)site24x7", "name": "Site24x7", "category": "Site Monitor" },
  { "regex": "(?i)feedfetcher", "name": "FeedFetcher", "category": "Feed Fetcher" },
  { "regex": "(?i)feedly", "name": "Feedly", "category": "Feed Fetcher" },
  { "regex": "(?i)curl/", "name": "curl", "category": "Service Agent" },
  { "regex": "(?i)wget/", "name": "Wget", "category": "Service Agent" },
  { "regex": "(?i)python-requests", "name": "Python Requests", "category": "Service Agent" },
  { "regex": "(?i)go-http-client", "name": "Go HTTP Client", "category": "Service Agent" },
  { "regex": "(?i)java/", "name": "Java", "category": "Service Agent" },
  { "regex": "(?i)apache-httpclient", "name": "Apache HttpClient", "category": "Service Agent" },
  { "regex": "(?i)headlesschrome", "name": "HeadlessChrome", "category": "Crawler" },
  { "regex": "(?i)\\b(?:bot|crawler|spider)\\b", "name": "Generic Bot", "category": "Generic" }
]
```

- [ ] **Step 2: Verify it is valid JSON**

Run: `python3 -m json.tool bots.json > /dev/null && echo OK`
Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add bots.json
git commit -m "feat: add curated bot ruleset data (bots.json)"
```

---

## Task 2: Bot detection engine

**Files:**
- Create: `bot.go`
- Test: `bot_test.go`

- [ ] **Step 1: Write the failing tests in `bot_test.go`**

```go
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
	// A real mobile device whose UA embeds "bot" inside an app name must NOT be a bot.
	ua := "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Mobile Safari/537.36 Botim/1.0"
	if got := detectBot(ua); got != nil {
		t.Errorf("detectBot(%q) = %+v, want nil (false positive)", ua, got)
	}
}

func TestLoadBotRules(t *testing.T) {
	if _, err := loadBotRules([]byte(`[{"regex":"(?i)googlebot","name":"Googlebot","category":"Search bot"}]`)); err != nil {
		t.Errorf("loadBotRules(valid) error = %v", err)
	}
	if _, err := loadBotRules([]byte(`not json`)); err == nil {
		t.Error("loadBotRules(bad json) error = nil, want error")
	}
	if _, err := loadBotRules([]byte(`[{"regex":"(","name":"x","category":"y"}]`)); err == nil {
		t.Error("loadBotRules(bad regex) error = nil, want error")
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
```

- [ ] **Step 2: Run tests to verify they fail (do not compile)**

Run: `go test ./... 2>&1 | tail -5`
Expected: build failure — `undefined: detectBot`, `undefined: loadBotRules`, `undefined: botRules`, `undefined: mustLoadBotRules`.

- [ ] **Step 3: Create `bot.go`**

```go
package devicedetector

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
)

//go:embed bots.json
var botData []byte

// BotInfo describes a bot identified from a User-Agent string.
type BotInfo struct {
	Name     string
	Category string
}

// botRule is a compiled bot-detection rule.
type botRule struct {
	re       *regexp.Regexp
	name     string
	category string
}

// botRuleJSON is the on-disk shape of a single rule in bots.json.
type botRuleJSON struct {
	Regex    string `json:"regex"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

// botRules holds the ordered, compiled bot rules loaded from bots.json.
// The order in the file is significant: specific bots come first, the
// generic catch-all last, so the most specific rule wins.
var botRules = mustLoadBotRules(botData)

// loadBotRules parses and compiles the embedded bot ruleset.
func loadBotRules(data []byte) ([]botRule, error) {
	var raw []botRuleJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse bots.json: %w", err)
	}
	rules := make([]botRule, 0, len(raw))
	for i, r := range raw {
		re, err := regexp.Compile(r.Regex)
		if err != nil {
			return nil, fmt.Errorf("bad bot regex at index %d (%q): %w", i, r.Regex, err)
		}
		rules = append(rules, botRule{re: re, name: r.Name, category: r.Category})
	}
	return rules, nil
}

// mustLoadBotRules is loadBotRules for author-controlled embedded data:
// a parse/compile failure is a programmer error, so it panics.
func mustLoadBotRules(data []byte) []botRule {
	rules, err := loadBotRules(data)
	if err != nil {
		panic("devicedetector: " + err.Error())
	}
	return rules
}

// detectBot returns the first matching bot rule's info for userAgent, or nil
// if no rule matches. Rules are case-insensitive and matched against the raw
// User-Agent in file order (first match wins).
func detectBot(userAgent string) *BotInfo {
	for _, rule := range botRules {
		if rule.re.MatchString(userAgent) {
			return &BotInfo{Name: rule.name, Category: rule.category}
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./... 2>&1 | tail -5`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add bot.go bot_test.go
git commit -m "feat: add data-driven bot detection engine"
```

---

## Task 3: Wire bot detection into Detect

**Files:**
- Modify: `devicedetector.go`
- Test: `devicedetector_test.go`

- [ ] **Step 1: Write the failing integration tests (append to `devicedetector_test.go`)**

```go
func TestDetectPopulatesBot(t *testing.T) {
	d := New()
	dev := d.Detect("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	if dev.Type != Bot {
		t.Errorf("Type = %v, want Bot", dev.Type)
	}
	if dev.Bot == nil {
		t.Fatal("Bot = nil, want non-nil")
	}
	if dev.Bot.Name != "Googlebot" || dev.Bot.Category != "Search bot" {
		t.Errorf("Bot = {%q,%q}, want {Googlebot, Search bot}", dev.Bot.Name, dev.Bot.Category)
	}
}

func TestDetectNonBotHasNilBot(t *testing.T) {
	d := New()
	// Mobile device with an app name containing "bot" — must be Mobile, not a bot.
	dev := d.Detect("Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Mobile Safari/537.36 Botim/1.0")
	if dev.Type != Mobile {
		t.Errorf("Type = %v, want Mobile", dev.Type)
	}
	if dev.Bot != nil {
		t.Errorf("Bot = %+v, want nil", dev.Bot)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail (do not compile)**

Run: `go test ./... 2>&1 | tail -5`
Expected: build failure — `dev.Bot undefined (type Device has no field or method Bot)`.

- [ ] **Step 3a: Add the `Bot` field to `Device` in `devicedetector.go`**

Replace:

```go
// Device holds the facts we extract from a single User-Agent string.
type Device struct {
	Type      DeviceType
	UserAgent string
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
}
```

- [ ] **Step 3b: Run bot detection first in `Detect`**

Replace:

```go
// Detect inspects a User-Agent string and returns the Device it describes.
func (d *Detector) Detect(userAgent string) Device {
	return Device{
		Type:      classify(strings.ToLower(userAgent)),
		UserAgent: userAgent,
	}
}
```

with:

```go
// Detect inspects a User-Agent string and returns the Device it describes.
// Bots are matched first (against the raw User-Agent); anything else is
// classified by device type.
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

- [ ] **Step 3c: Remove the bot case from `classify`**

Replace:

```go
// classify maps a lower-cased User-Agent string to a DeviceType.
//
// Order is significant: User-Agent strings overlap (an iPad UA contains
// "mobile", a bot UA contains "android"/"mozilla"), so the most
// specific/overriding categories are tested first — bot, then Android
// (split into phone vs. tablet), then iPad/tablet, then other mobile,
// then desktop.
func classify(ua string) DeviceType {
	switch {
	case containsAny(ua, "bot", "spider", "crawl"):
		return Bot
	case strings.Contains(ua, "android"):
```

with:

```go
// classify maps a lower-cased User-Agent string to a DeviceType. Bot
// detection happens earlier in Detect, so this handles device types only.
//
// Order is significant: User-Agent strings overlap (an iPad UA contains
// "mobile"), so the most specific/overriding categories are tested first —
// Android (split into phone vs. tablet), then iPad/tablet, then other
// mobile, then desktop.
func classify(ua string) DeviceType {
	switch {
	case strings.Contains(ua, "android"):
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./... 2>&1 | tail -5`
Expected: PASS (all tests, including the pre-existing `TestDetect` cases).

- [ ] **Step 5: Commit**

```bash
git add devicedetector.go devicedetector_test.go
git commit -m "feat: report bot name/category from Detect; drop substring bot match"
```

---

## Task 4: Verify coverage and update docs

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Confirm formatting, vet, tests, and coverage**

Run:
```bash
gofmt -l . && go vet ./... && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
```
Expected: `gofmt` prints nothing; vet clean; tests PASS; total coverage `> 90%`.

- [ ] **Step 2: Document bot detection in `README.md`**

Add this section immediately before the `## Limitations` section:

```markdown
## Bot detection

When the User-Agent matches a known bot, `Detect` sets `Type` to `Bot` and
populates `Device.Bot` with the bot's name and category:

```go
device := d.Detect("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
if device.Bot != nil {
	fmt.Println(device.Bot.Name, "-", device.Bot.Category) // Googlebot - Search bot
}
```

Bots are matched against an embedded, curated ruleset (`bots.json`) of
ordered case-insensitive regexes, first-match-wins. The list is **not
exhaustive** — it covers common search engines, social crawlers, SEO and
monitoring tools, AI crawlers, and a generic catch-all.
```

- [ ] **Step 3: Replace the bot limitation bullet in `README.md`**

In the `## Limitations` section, replace:

```markdown
- **Bot detection** keys on the substrings `bot`/`spider`/`crawl`, so it can
  both over-match (a device/app token containing "bot") and miss tokenless
  crawlers (e.g. `facebookexternalhit`, `WhatsApp`).
```

with:

```markdown
- **Bot detection** uses a curated, non-exhaustive ruleset, so bots not in
  `bots.json` are not identified and fall through to device classification.
```

- [ ] **Step 4: Update the architecture notes in `CLAUDE.md`**

In the `## Architecture` section, replace the opening line:

```markdown
The core flow is `New() -> Detector.Detect(userAgent) -> Device`:
```

with:

```markdown
The core flow is `New() -> Detector.Detect(userAgent) -> Device`:

- **Bot detection runs first** (`bot.go`): `detectBot` matches the raw UA
  against an embedded, ordered ruleset compiled from `bots.json`
  (case-insensitive RE2 regexes, first-match-wins). A hit sets `Type == Bot`
  and populates `Device.Bot`. `bots.json` is our own curated data — matomo's
  architecture, not its LGPL-licensed data. To add a bot, add a `{regex, name,
  category}` entry (specific rules before the generic catch-all). Bad
  data/regex panics at init; a test compiles every rule.
```

- [ ] **Step 5: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: document bot detection (README, CLAUDE.md)"
```

---

## After the plan

Run the repo's PR merge gate from `CLAUDE.md`: push the branch, open a PR, then
`/simplify` → PR review → address bot review comments → merge once green.

## Self-Review notes

- **Spec coverage:** engine (Task 2), embedded JSON data (Task 1), `Bot`
  API field (Task 3), detection flow / removal of substring case (Task 3),
  error handling via panic + `loadBotRules` error tests (Task 2), false-positive
  guard + known-bot + integrity tests (Tasks 2–3), coverage + docs (Task 4).
  All spec sections map to a task.
- **Type consistency:** `BotInfo{Name, Category}`, `botRule{re, name,
  category}`, `loadBotRules`/`mustLoadBotRules`, `detectBot`, and `Device.Bot`
  are used identically across every task.
- **Placeholders:** none — every code/data step is complete.
