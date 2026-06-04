package devicedetector

import _ "embed"

//go:embed oss.json
var osData []byte

// OSInfo describes the operating system parsed from a User-Agent string.
type OSInfo struct {
	Name    string
	Version string
}

// osRules holds the ordered, compiled OS rules from oss.json. Order is
// significant (e.g. iOS before macOS, since an iOS UA contains "Mac OS X").
var osRules = mustLoadVersionedRules(osData)

// detectOS returns the OS for userAgent, or nil if no rule matches.
func detectOS(userAgent string) *OSInfo {
	if name, version, ok := matchVersioned(osRules, userAgent); ok {
		return &OSInfo{Name: name, Version: version}
	}
	return nil
}
