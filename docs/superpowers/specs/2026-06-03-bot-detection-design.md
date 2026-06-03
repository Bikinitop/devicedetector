# Data-driven bot detection

**Date:** 2026-06-03
**Status:** Approved (design)
**Scope:** First increment of adopting matomo-org/device-detector ideas into `devicedetector`.

## Background

`devicedetector` currently classifies a User-Agent into a `DeviceType`
(`Unknown`/`Desktop`/`Mobile`/`Tablet`/`Bot`) via flat substring matching. A
prior code review CONFIRMED a defect in bot handling: matching the bare
substrings `bot`/`spider`/`crawl` both **over-matches** (a device/app token
containing "bot", e.g. "Botim"/"Botswana", is misclassified as `Bot` and
overrides device detection) and **under-matches** (tokenless crawlers such as
`facebookexternalhit` and `WhatsApp` are never identified).

[matomo-org/device-detector](https://github.com/matomo-org/device-detector) is
the canonical reference. It is **LGPL-3.0**, with its detection data held in
large YAML regex files (`regexes/bots.yml` is ~145 KB). We are **not** copying
its data — that would make this library a derivative work bound by LGPL-3.0.
We adopt only its *architecture* (data-driven, ordered `{regex, name}` rules,
first-match-wins) and author our own curated data, keeping this library
permissively licensed and zero-dependency.

## Goals

- Replace substring bot matching with a data-driven, ordered regex ruleset.
- Identify well-known bots accurately, including tokenless crawlers.
- Expose the matched bot's name and category.
- Retire the CONFIRMED false-positive / false-negative review bug.
- Establish an engine that future dimensions (OS, browser) can reuse.

## Non-goals (YAGNI)

- Bot version, URL, or producer metadata.
- OS, browser/client, device brand, or device model detection.
- Exhaustive bot coverage. The curated set (~40–80 entries) is documented as
  non-exhaustive.
- Any external dependency or copying of matomo's data files.

## Architecture

A new `bot.go` plus an embedded `bots.json` data file.

```
Detect(userAgent)
  ├─ detectBot(userAgent)              // raw UA, case-insensitive regexes, ordered
  │     └─ first matching rule ─► Device{Type: Bot, UserAgent: ua, Bot: &BotInfo{Name, Category}}
  └─ no bot match ─► Device{Type: classify(strings.ToLower(ua)), UserAgent: ua, Bot: nil}
```

The existing `bot`/`spider`/`crawl` case is **removed** from `classify`;
`classify` is reduced to device-type rules (android → ipad/tablet → mobile →
desktop → unknown). Bot detection runs before `classify`, preserving the
"bot wins over device" priority.

## Data model

`bots.json` — an array of objects, authored by us:

```json
[
  { "regex": "(?i)googlebot",        "name": "Googlebot",          "category": "Search bot" },
  { "regex": "(?i)facebookexternalhit", "name": "Facebook",        "category": "Social Media Agent" },
  { "regex": "(?i)whatsapp",         "name": "WhatsApp",           "category": "Social Media Agent" }
]
```

- Curated ~40–80 common bots across categories: Search bot, Social Media Agent,
  SEO/Crawler, Site Monitor, Feed Fetcher, and a small set of generic
  catch-alls (e.g. a trailing `(?i)\bbot\b|crawler|spider` rule) ordered LAST so
  specific bots win.
- Regexes are **RE2-compatible** (no backreferences/lookarounds) and
  case-insensitive via the `(?i)` flag.
- Embedded with `//go:embed bots.json` and parsed once.

Loaded into:

```go
type botRule struct {
	re       *regexp.Regexp
	name     string
	category string
}
```

## Public API (additive, backward-compatible)

```go
type Device struct {
	Type      DeviceType
	UserAgent string
	Bot       *BotInfo // non-nil only when Type == Bot
}

type BotInfo struct {
	Name     string
	Category string
}
```

Existing fields and behavior are unchanged. `Bot` is `nil` for non-bots, so
existing callers are unaffected; `Type` still reports `Bot`.

## Detection logic

- `detectBot(userAgent string) *BotInfo`: iterate `botRules` in order, return
  the first rule whose `re.MatchString(userAgent)` is true (matched against the
  **raw** UA; `(?i)` handles case). Return `nil` if none match.
- Matching the raw UA (not the lowercased copy) keeps the bot layer independent
  of `classify`'s lowercasing contract.

## Error handling

Embedded data is author-controlled, so malformed JSON or an uncompilable regex
is a programmer error. The package **panics at `init`** (fail-fast, mirroring
`regexp.MustCompile`). A test parses+compiles every rule so any bad entry fails
in CI, never in production. `Detect` itself never returns an error.

## Testing (TDD, red → green → refactor)

Table-driven tests in `bot_test.go` (and additions to `devicedetector_test.go`):

- **Known bots** → `Type == Bot`, correct `Bot.Name` and `Bot.Category`:
  Googlebot, Bingbot, DuckDuckBot, facebookexternalhit, WhatsApp, Slackbot,
  Twitterbot, AhrefsBot (and a few more spanning each category).
- **False-positive guard** → the "Botim"/"Botswana"-style device UA classifies
  as a device (e.g. `Mobile`) with `Bot == nil`.
- **Non-bots** → existing device cases keep their `DeviceType` and `Bot == nil`.
- **Rule integrity** → all embedded rules parse and compile; the set is
  non-empty.

Coverage must stay > 90%.

## Future work (out of scope here)

- OS detection ruleset (`oss.json` + parser) using the same engine.
- Browser/client detection (`browsers.json`).
- These reuse the embedded-JSON + ordered-regex pattern established here.
