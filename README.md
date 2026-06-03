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

## Development

```bash
go test ./...   # run tests
go vet ./...    # static checks
```
