package devicedetector

import "regexp"

// appleModelRe matches Apple's generic device word. Apple does not expose the
// specific model in the User-Agent.
var appleModelRe = regexp.MustCompile(`(?i)(iphone|ipad|ipod)`)

// androidModelRe captures the raw Android model token between the OS version
// and the next "Build" or ")".
var androidModelRe = regexp.MustCompile(`(?i)android [\w.]+; ?([^;)]+?)(?: build|\))`)

// detectModel returns the raw device model for userAgent, or "" if none
// matches. The model is returned verbatim (e.g. "Pixel 7", "SM-G991B"); it is
// not normalized or mapped to a marketing name. Apple reports only the device
// word (iPhone/iPad/iPod).
func detectModel(userAgent string) string {
	if m := appleModelRe.FindStringSubmatch(userAgent); m != nil {
		return m[1]
	}
	if m := androidModelRe.FindStringSubmatch(userAgent); m != nil {
		return m[1]
	}
	return ""
}
