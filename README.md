# devicedetector

[![CI](https://github.com/Bikinitop/devicedetector/actions/workflows/ci.yml/badge.svg)](https://github.com/Bikinitop/devicedetector/actions/workflows/ci.yml)

A small Go library for detecting device information from an HTTP `User-Agent` string.

## Install

```bash
go get github.com/Bikinitop/devicedetector
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/Bikinitop/devicedetector"
)

func main() {
	d := devicedetector.New()
	device := d.Detect("Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) ...")
	fmt.Printf("%+v\n", device)
}
```

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

## Limitations

Detection is intentionally minimal: device typing is a heuristic **substring
match** on the User-Agent, and bots are matched against a curated regex
ruleset. Some real-world cases are out of scope for now (tracked as future
work):

- **iPadOS 13+ in desktop mode** is reported as `Desktop` — Apple sends a UA
  byte-identical to desktop Safari, so an iPad cannot be distinguished from a
  Mac by User-Agent alone (it needs client-side signals such as
  `navigator.maxTouchPoints`).
- **Bot detection** uses a curated, non-exhaustive ruleset, so bots not in
  `bots.json` are not identified and fall through to device classification.

## Development

```bash
go test ./...   # run tests
go vet ./...    # static checks
```
