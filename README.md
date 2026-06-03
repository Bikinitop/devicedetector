# devicedetector

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

## Limitations

Detection is a fast, heuristic **substring match** on the User-Agent. It is
intentionally minimal, which means some real-world cases are out of scope for
now (tracked as future work):

- **iPadOS 13+ in desktop mode** is reported as `Desktop` — Apple sends a UA
  byte-identical to desktop Safari, so an iPad cannot be distinguished from a
  Mac by User-Agent alone (it needs client-side signals such as
  `navigator.maxTouchPoints`).
- **Bot detection** keys on the substrings `bot`/`spider`/`crawl`, so it can
  both over-match (a device/app token containing "bot") and miss tokenless
  crawlers (e.g. `facebookexternalhit`, `WhatsApp`).

## Development

```bash
go test ./...   # run tests
go vet ./...    # static checks
```
