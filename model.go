package devicedetector

import (
	"regexp"
	"strings"
)

// appleModelRe matches Apple's generic device word. Apple does not expose the
// specific model in the User-Agent. The token is left-anchored so it does not
// match inside an unrelated substring (e.g. "Lipad").
var appleModelRe = regexp.MustCompile(`(?i)(?:^|[^a-z])(iphone|ipad|ipod)`)

// androidModelRe captures the raw Android model token between the OS version
// and the next "Build" or ")". An optional locale token (e.g. "en-us; ") is
// skipped, since older Android UAs place it before the model.
var androidModelRe = regexp.MustCompile(`(?i)android [\w.]+;\s*(?:[a-z]{2}-[a-z]{2};\s*)?([^;)]+?)(?: build|\))`)

// detectModel returns the raw device model for userAgent, or "" if none
// matches. The model is returned verbatim (e.g. "Pixel 7", "SM-G991B"); it is
// not normalized or mapped to a marketing name. Apple reports only the device
// word (iPhone/iPad/iPod).
func detectModel(userAgent string) string {
	if m := appleModelRe.FindStringSubmatch(userAgent); m != nil {
		return m[1]
	}
	if m := androidModelRe.FindStringSubmatch(userAgent); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}
