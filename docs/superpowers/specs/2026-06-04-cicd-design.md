# CI/CD with GitHub Actions + release-please

**Date:** 2026-06-04
**Status:** Approved (design)
**Scope:** Add continuous integration and automated release/tagging to the
`devicedetector` Go library.

## Background

The repository has no CI. The quality rules in `CLAUDE.md` (tests pass, `go
vet` clean, `gofmt` clean, coverage > 90%) are currently enforced only by the
agent running them locally — there is no machine gate, so prior PRs were
"vacuously green" with respect to CI. The repo already uses conventional-commit
messages (`feat:`, `fix:`, `docs:`, `chore:`), which enables commit-driven
release automation.

## Goals

- Verify every pull request and every push to `main` automatically.
- Machine-enforce the existing quality bar: build, test (with `-race`), `gofmt`,
  `go vet`, `golangci-lint`, and the **> 90% coverage gate**.
- Automate versioning and releases from conventional commits, gated by a
  human-merged "Release PR".
- Keep releases publishing to `pkg.go.dev` via standard `vX.Y.Z` tags.

## Non-goals (YAGNI)

- No multi-OS or multi-Go-version test matrix (single version from `go.mod`).
- No artifact builds (it is a library — nothing to compile/ship).
- No branch-protection configuration (a repo setting, not a file; noted as an
  optional manual follow-up).
- No external CI services, secrets, or runtime dependencies beyond the built-in
  `GITHUB_TOKEN`.

## Components

### 1. `.github/workflows/ci.yml`

Triggers: `pull_request` and `push` to `main`. Top-level `permissions:
contents: read`. Two jobs run in parallel:

- **test**
  - `actions/checkout`
  - `actions/setup-go` with `go-version-file: go.mod` (single source of truth
    for the Go version).
  - `go test -race -coverprofile=coverage.out ./...`
  - Coverage gate: parse the total from `go tool cover -func=coverage.out` and
    fail if it is below the `90` threshold.
- **lint**
  - `actions/checkout`, `actions/setup-go` (`go-version-file: go.mod`).
  - `test -z "$(gofmt -l .)"` (fail if any file is unformatted, printing the
    offenders).
  - `go vet ./...`.
  - `golangci-lint run` via `golangci/golangci-lint-action` with a **pinned**
    linter version.

### 2. `.golangci.yml`

Minimal curated config enabling a sensible default linter set (`errcheck`,
`govet`, `ineffassign`, `staticcheck`, `unused`, `gofmt`). Kept light to avoid
noise. The linter version is pinned in the action so a future linter release
cannot spontaneously break `main`.

### 3. `.github/workflows/release-please.yml`

Triggers: `push` to `main`. Permissions: `contents: write`, `pull-requests:
write`. Runs `googleapis/release-please-action` (manifest-driven). It reads
conventional commits since the last release and maintains a "Release PR" that
updates `CHANGELOG.md`; merging that PR creates the `vX.Y.Z` git tag and a
GitHub Release. No source version file is required for the Go release type;
release state lives in the manifest.

### 4. `release-please-config.json` + `.release-please-manifest.json`

- `release-please-config.json`: a manifest config with package `"."` set to
  `release-type: go` and `include-component-in-tag: false` (tags are bare
  `vX.Y.Z`).
- `.release-please-manifest.json`: `{ ".": "0.0.0" }` — seed state. The
  accumulated `feat:`/`fix:` history drives the first computed release
  (a `feat:` present ⇒ first release is `v0.1.0`).

## Interaction with the existing PR gate

`ci.yml` runs on all pull requests, so from this PR onward the "CI green" step
of the `CLAUDE.md` merge gate is real rather than vacuous — including on
release-please's own Release PRs (which only touch `CHANGELOG.md`/manifest and
still pass tests).

## Verification

- The introducing PR is itself checked by `ci.yml` (PRs trigger it), so the
  green run is observable on the PR.
- Before pushing, run locally and confirm clean: `gofmt -l .`, `go vet ./...`,
  the coverage-gate script, and `golangci-lint run`. **Any new golangci-lint
  findings are fixed in the same PR** so it lands green.
- release-please tagging cannot be fully exercised until the workflow is on
  `main`; success criterion is observing it open a Release PR after merge.

## Risks

- **golangci-lint may surface findings `go vet` does not.** Mitigation: run it
  locally first and fix in this PR; pin the version.
- **Action version drift.** Mitigation: pin major versions of all third-party
  actions (`setup-go`, `golangci-lint-action`, `release-please-action`).

## Future work (out of scope here)

- Optional branch-protection rule requiring the `test` and `lint` checks before
  merge to `main`.
- Test matrix across Go versions if backward compatibility becomes a concern.
