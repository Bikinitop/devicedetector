# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`devicedetector` is a Go library (module `github.com/Bikinitop/devicedetector`) that parses HTTP `User-Agent` strings and reports device information. It is an importable package with no `main` — the public API lives in the root package.

## Commands

```bash
go test ./...                          # run all tests
go test -run TestDetect/iphone_is_mobile ./...   # run a single test case (use the subtest name after the slash)
go test -cover ./...                   # show coverage summary
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out   # per-function coverage
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out   # coverage in browser
go vet ./...                           # static analysis
gofmt -l .                             # list files needing formatting (must be empty)
```

## Architecture

The core flow is `New() -> Detector.Detect(userAgent) -> Device`:

- **Bot detection runs first** (`bot.go`): `detectBot` matches the raw UA
  against an embedded, ordered ruleset compiled from `bots.json`
  (case-insensitive RE2 regexes, first-match-wins). A hit sets `Type == Bot`
  and populates `Device.Bot`. `bots.json` is our own curated data — matomo's
  architecture, not its LGPL-licensed data. To add a bot, add a `{regex, name,
  category}` entry (specific rules before the generic catch-all). Bad
  data/regex panics at init; a test compiles every rule.

- **OS and browser detection** (`os.go`, `browser.go`) run for non-bot UAs via a shared versioned matcher (`match.go`): embedded ordered regex rulesets (`oss.json`, `browsers.json`) whose capture group 1 is the version (`_`→`.` normalized). `Detect` sets `Device.OS`/`Device.Browser` (nil for bots / no match). Ordering is correctness-critical (iOS before macOS; browser derivatives before Chrome/Firefox; Safari after the Chrome family). To add a rule, add a `{regex, name}` entry with a version capture group; bad data/regex panics at init and a test compiles every rule.

- **Brand and model detection** (`brand.go`, `model.go`) run for non-bot UAs: `detectBrand` matches a curated `brands.json` token ruleset (via the shared `matchFirst`, the raw-capture sibling of `matchVersioned`); `detectModel` uses two patterns (Apple device word; the Android model token between the OS version and `Build`/`)`). `Detect` sets `Device.Brand`/`Device.Model` (`""` for bots / no match). The model is raw — no marketing-name mapping.

- `Detector` is an empty, immutable struct and is therefore safe for concurrent use — a single instance can be shared across goroutines. Do not add mutable state to it without revisiting that concurrency contract.
- `Detect` lower-cases the User-Agent once, then delegates the actual categorization to the unexported `classify` helper. Keep `classify` operating on already-lowercased input so matching logic stays simple.
- `DeviceType` is an `int`-backed enum implementing `Stringer`. When adding a new category, add the constant **and** its `String()` case together.
- Classification rule **ordering is correctness-critical**: more specific/overriding categories must be checked first (bot → android → iPad/tablet → other mobile → desktop → unknown), because User-Agent strings overlap (e.g. an iPad UA contains both tablet and mobile-like tokens).
- **Android is split on the `mobile` token**: Android phones carry `mobile` in their UA, tablets omit it — this is the one reliable phone/tablet signal Android exposes, so it gets a dedicated branch ahead of the generic tablet/mobile cases.

## CI/CD

- **CI** (`.github/workflows/ci.yml`) runs on every pull request and push to
  `main`: a `test` job (`go test -race` + the > 90% coverage gate) and a `lint`
  job (`gofmt`, `go vet`, `golangci-lint`). This is the machine enforcement of
  the quality rules below.
- **Releases** are automated by release-please (`.github/workflows/release-please.yml`):
  conventional commits on `main` drive a "Release PR" that, when merged, tags
  `vX.Y.Z` and creates a GitHub Release. Keep using conventional-commit
  messages (`feat:`, `fix:`, `docs:`, `chore:`) so versioning stays correct.

## Workflow (required)

These rules are mandatory for all changes in this repo:

- **TDD, strictly red → green → refactor.** Write a failing test first, confirm it fails (red), make it pass with the minimum change (green), then refactor. Tests double as the spec — see `devicedetector_test.go` for the table-driven pattern to follow.
- **Coverage must stay above 90%.** Verify with `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` before opening a PR.
- **Never commit directly to `main`.** Always create a feature branch and do the work there.
- **Every change ships as a PR.** After implementation, open a PR rather than merging locally.
- **PR gate before merge** — a PR must pass all of these, in order, before it can be merged:
  1. `/simplify` — apply simplification/cleanup to the change.
  2. PR review (`/code-review` or `/review`).
  3. Address bot review comments — read the PR's bot-generated review comments, fix the issues they raise, and push the fixes.
  Only merge once all three are done and CI is green.
