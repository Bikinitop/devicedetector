# OS + browser detection (name + version)

**Date:** 2026-06-04
**Status:** Approved (design)
**Scope:** Add operating-system and browser/client detection (name + version) to
`devicedetector`, reusing and extending the data-driven bot-detection engine.

## Background

`devicedetector` detects device type (`classify`) and bots (`bot.go`:
data-driven, ordered, case-insensitive regexes, first-match-wins). The next
increment adds two more dimensions — OS and browser — each reporting a **name
and version**. The new engine capability versus bots is **version extraction**
via a regex capture group (bots have fixed names; `Chrome/120.0` must yield
`120.0`).

This follows matomo-org/device-detector's architecture (separate OS and client
datasets) but uses our own curated, permissively-licensed data — matomo's YAML
is LGPL and is not copied.

## Goals

- Detect OS name + version and browser name + version from a User-Agent.
- Expose them as flat, independently-nil-able fields on `Device`.
- Reuse the bot engine's data-driven pattern; factor out shared loading.
- Keep the library minimal and zero-dependency; maintain > 90% coverage.

## Non-goals (YAGNI)

- No browser layout engine, OS platform/bitness, device brand, or model.
- No Windows NT-version → marketing-name mapping (report the NT version as the
  UA states it).
- Bots are not assigned an OS or browser.
- Curated subsets, not exhaustive coverage (documented).
- No external dependencies; no copying of matomo data.

## Architecture

Two new dimensions sharing one matcher:

```
Detect(userAgent)
  ├─ detectBot(ua) != nil ─► Device{Type: Bot, Bot: …, OS: nil, Browser: nil}
  └─ otherwise ─► Device{
        Type:    classify(strings.ToLower(ua)),
        OS:      detectOS(ua),       // *OSInfo or nil
        Browser: detectBrowser(ua),  // *BrowserInfo or nil
     }
```

### Shared matcher — `match.go`

- `compileInsensitive(pattern string) (*regexp.Regexp, error)` — applies the
  central `(?i)` flag, rejects an empty pattern, compiles. Used by the OS and
  browser loaders **and** retrofitted into `bot.go`'s loader to remove the
  duplicated compile/`(?i)`/validate logic (light, test-covered refactor).
- `versionedRule struct { re *regexp.Regexp; name string }` — on-disk shape
  `{ "regex": "...", "name": "..." }`, where the regex has an optional capture
  group 1 holding the version.
- `loadVersionedRules(data []byte) ([]versionedRule, error)` + a
  `mustLoadVersionedRules` panic wrapper (mirrors the bot loader: rejects empty
  regex/name, panics on bad embedded data).
- `matchVersioned(rules []versionedRule, ua string) (name, version string, ok bool)`
  — first rule whose `re` matches; `version` = submatch group 1 if the regex
  has one (else `""`), with `_` normalized to `.` (for iOS/macOS UA tokens like
  `17_0`); `name` = the rule's name.

### OS — `os.go` + `oss.json`

- `OSInfo struct { Name, Version string }`.
- `//go:embed oss.json`; `detectOS(ua string) *OSInfo` (nil on no match).
- Curated rules, ordered specific-first: iOS, Android, macOS, Windows, ChromeOS,
  Linux. Version is the capture group, as the UA reports it (Windows = NT
  version, e.g. `10.0`).

### Browser — `browser.go` + `browsers.json`

- `BrowserInfo struct { Name, Version string }`.
- `//go:embed browsers.json`; `detectBrowser(ua string) *BrowserInfo` (nil on no
  match).
- Ordering is correctness-critical because UAs nest: **Edge → Opera → Samsung
  Internet → CriOS (iOS Chrome) → Chrome → Safari → Firefox**. Edge and Chrome
  both contain `Safari`; Chrome-derivatives contain `Chrome`; real Safari is
  matched by a `Version/<v> … Safari` pattern only after the Chrome family.

## Public API (additive, backward-compatible)

```go
type OSInfo struct {
	Name    string
	Version string
}

type BrowserInfo struct {
	Name    string
	Version string
}

type Device struct {
	Type      DeviceType
	UserAgent string
	Bot       *BotInfo
	OS        *OSInfo      // non-nil when an OS rule matches (nil for bots)
	Browser   *BrowserInfo // non-nil when a browser rule matches (nil for bots)
}
```

Existing fields/behavior are unchanged; `OS`/`Browser` are additive and `nil`
for bots or when nothing matches.

## Error handling

Embedded data is author-controlled: malformed JSON or an uncompilable regex
panics at `init` (fail-fast, like the bot loader and `regexp.MustCompile`).
Tests parse + compile every rule so a bad entry fails in CI. `Detect` never
returns an error.

## Testing (TDD, red → green → refactor)

Table-driven tests across real User-Agents:

- **OS**: iPhone iOS (`17.0`), Android (`13`), macOS (`10.15.7`), Windows
  (`10.0`), ChromeOS, Linux.
- **Browser**: Windows Chrome, macOS Safari, iOS Safari, iOS Chrome (CriOS),
  Edge, Firefox, Samsung Internet, Opera.
- **Ordering guards**: a Chrome UA → `Chrome` (not Safari); an Edge UA → `Edge`
  (not Chrome).
- **Bots**: a bot UA → `OS == nil` and `Browser == nil`.
- **No match**: an unrecognized UA → `nil`.
- **Rule integrity**: all embedded OS and browser rules parse and compile; sets
  are non-empty. `loadVersionedRules` rejects empty regex/name and bad JSON;
  `mustLoadVersionedRules` panics on bad data.

Coverage must stay > 90%.

## Future work (out of scope here)

- Device brand/model (matomo's largest dataset).
- Browser engine / OS platform fields.
- Windows NT-version → marketing-name mapping.
