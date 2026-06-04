package devicedetector

import _ "embed"

//go:embed browsers.json
var browserData []byte

// BrowserInfo describes the browser parsed from a User-Agent string.
type BrowserInfo struct {
	Name    string
	Version string
}

// browserRules holds the ordered, compiled browser rules from browsers.json.
// Order is significant: Chrome-derivatives (Edge/Opera/Samsung/CriOS) precede
// Chrome, and real Safari is matched after the Chrome family, because these
// User-Agents nest each other's tokens.
var browserRules = mustLoadVersionedRules(browserData)

// detectBrowser returns the browser for userAgent, or nil if no rule matches.
func detectBrowser(userAgent string) *BrowserInfo {
	if name, version, ok := matchVersioned(browserRules, userAgent); ok {
		return &BrowserInfo{Name: name, Version: version}
	}
	return nil
}
