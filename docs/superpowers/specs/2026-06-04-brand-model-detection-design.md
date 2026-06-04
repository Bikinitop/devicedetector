# Device brand + raw model detection

**Date:** 2026-06-04
**Status:** Approved (design)
**Scope:** Add device brand and raw model detection to `devicedetector`,
reusing the shared versioned-rule matcher.

## Background

`devicedetector` (v0.2.0) detects device type, bot, OS, and browser via
data-driven, ordered, case-insensitive regex rulesets and a shared matcher
(`match.go`). The next dimension is device **brand** and **model**.

matomo-org/device-detector's value here is `mobiles.yml` (~1.3 MB) mapping model
*codes* to *marketing names* (`SM-G991B` → "Galaxy S21"). We deliberately do not
replicate that. We detect the brand from UA tokens and extract the **raw** model
identifier the UA already carries (`Pixel 7`, `SM-G991B`), with no marketing-name
mapping. Apple hides the specific model, so for Apple the model is the generic
device word (`iPhone`/`iPad`/`iPod`).

## Goals

- Detect device brand from a curated set of UA tokens.
- Extract the raw model identifier (Android model token; Apple device word).
- Expose both as flat `Device.Brand` / `Device.Model` strings (`""` = unknown).
- Reuse the existing matcher; keep the library minimal and zero-dependency;
  maintain > 90% coverage.

## Non-goals (YAGNI)

- No marketing-name mapping (`SM-G991B` stays `SM-G991B`).
- No specific Apple model (`iPhone`, never "iPhone 15").
- No desktop PC vendor (UAs do not carry it).
- Curated brand subset, not exhaustive.
- Bots are assigned no brand/model.
- No external dependencies; no copying of matomo data.

## Architecture

Two independent string-yielding detectors, wired into `Detect` for non-bot UAs.

### Shared matcher change — `match.go`

- Add `matchFirst(rules []versionedRule, userAgent string) (name, capture string, ok bool)`
  — the ordered, first-match-wins loop returning the **raw** capture group 1
  (no normalization).
- Refactor `matchVersioned` to call `matchFirst` and apply the `_`→`.` version
  normalization to the returned capture (small, test-covered refactor; behavior
  for OS/browser is unchanged). Model captures must stay raw, so they use
  `matchFirst` directly.

### Brand — `brand.go` + `brands.json`

- `//go:embed brands.json`; `brandRules = mustLoadVersionedRules(brandData)`.
- `detectBrand(userAgent string) string` — `name, _, ok := matchFirst(brandRules, ua)`;
  returns `name` or `""`.
- Curated, ordered `{regex, name}` rules (~15–20 brands): Apple
  (`iphone|ipad|ipod|macintosh`), Samsung (`sm-[a-z0-9]+|samsung`), Google
  (`pixel`), Xiaomi (`redmi|poco|xiaomi`), Huawei (`huawei|honor`), OnePlus,
  Oppo, Vivo, Motorola (`moto|xt\d`), Sony (`xperia|sony`), Nokia, etc. Short or
  ambiguous tokens are anchored (`(?:^|[^a-z])…`) to avoid substring
  false-positives, consistent with the browser rules.

### Model — `model.go`

`detectModel(userAgent string) string` using two compiled patterns (it is
genuinely two cases):

- Apple device word: `(iphone|ipad|ipod)` → the captured word (`iPhone`/etc.).
- Android model token: `android [\w.]+; ?([^;)]+?)(?: build|\))` → the model
  between the Android version and the next `Build`/`)` (e.g. `Pixel 7`,
  `SM-G991B`; the frozen `K` on reduced UAs).

Apple is tried first; returns `""` if neither matches. The model is returned
verbatim (raw); it is not normalized or mapped.

## Public API (additive, backward-compatible)

```go
type Device struct {
	Type      DeviceType
	UserAgent string
	Bot       *BotInfo
	OS        *OSInfo
	Browser   *BrowserInfo
	Brand     string // "" if unknown
	Model     string // "" if unknown
}
```

Existing fields/behavior unchanged. `Brand`/`Model` are `""` for bots or when
nothing matches. They are independent: a known model with unknown brand, or a
brand with no model (e.g. Mac → `Apple`/`""`), are both representable.

## Detection flow

`Detect`: bot → `Brand`/`Model` stay `""`. Otherwise set
`Brand: detectBrand(ua)`, `Model: detectModel(ua)` alongside the existing
`Type`/`OS`/`Browser`.

## Error handling

Embedded `brands.json` is author-controlled: malformed JSON or an uncompilable
regex panics at `init` (fail-fast). The two model patterns are compiled at init
the same way. Tests parse + compile every rule. `Detect` never returns an error.

## Testing (TDD, red → green → refactor)

Table-driven across real User-Agents:

- iPhone → `Apple`/`iPhone`; iPad → `Apple`/`iPad`.
- Google Pixel → `Google`/`Pixel 7`.
- Samsung `SM-G991B` → `Samsung`/`SM-G991B`.
- Xiaomi/Huawei/OnePlus brand cases (model raw).
- macOS → `Apple`/`""` (brand without model).
- Windows desktop → `""`/`""` (no PC vendor/model).
- Bot UA → `Brand == ""` and `Model == ""`.
- Brand anchoring guard (a token containing a brand substring inside another
  word is not misattributed).
- `matchFirst` returns the raw capture; `matchVersioned` still normalizes
  `_`→`.` (regression guard for OS/browser).

Coverage must stay > 90%.

## Future work (out of scope here)

- Marketing-name mapping (model code → friendly name).
- Broader brand/model coverage.
- Device-type refinement using brand/model signals.
