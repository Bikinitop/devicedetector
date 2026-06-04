// Package devicedetector parses HTTP User-Agent strings and reports
// information about the originating device.
package devicedetector

import "strings"

// DeviceType is the broad category a User-Agent belongs to.
type DeviceType int

const (
	// Unknown is returned when no rule matches the User-Agent.
	Unknown DeviceType = iota
	Desktop
	Mobile
	Tablet
	Bot
)

// String renders a DeviceType for printing and logging.
func (t DeviceType) String() string {
	switch t {
	case Desktop:
		return "desktop"
	case Mobile:
		return "mobile"
	case Tablet:
		return "tablet"
	case Bot:
		return "bot"
	default:
		return "unknown"
	}
}

// Device holds the facts we extract from a single User-Agent string.
type Device struct {
	Type      DeviceType
	UserAgent string
	// Bot is non-nil only when Type == Bot; it carries the identified bot's
	// name and category.
	Bot *BotInfo
	// OS and Browser are populated for non-bot User-Agents when a rule
	// matches; they are nil for bots or when nothing matches.
	OS      *OSInfo
	Browser *BrowserInfo
}

// Detector parses User-Agent strings. Create one with New and reuse it;
// it is safe for concurrent use because it holds no mutable state.
type Detector struct{}

// New returns a ready-to-use Detector.
func New() *Detector {
	return &Detector{}
}

// Detect inspects a User-Agent string and returns the Device it describes.
// Bots are matched first (against the raw User-Agent); anything else is
// classified by device type.
func (d *Detector) Detect(userAgent string) Device {
	if bot := detectBot(userAgent); bot != nil {
		return Device{Type: Bot, UserAgent: userAgent, Bot: bot}
	}
	return Device{
		Type:      classify(strings.ToLower(userAgent)),
		UserAgent: userAgent,
		OS:        detectOS(userAgent),
		Browser:   detectBrowser(userAgent),
	}
}

// classify maps a lower-cased User-Agent string to a DeviceType. Bot
// detection happens earlier in Detect, so this handles device types only.
//
// Order is significant: User-Agent strings overlap (an iPad UA contains
// "mobile"), so the most specific/overriding categories are tested first —
// Android (split into phone vs. tablet), then iPad/tablet, then other
// mobile, then desktop.
func classify(ua string) DeviceType {
	switch {
	case strings.Contains(ua, "android"):
		// Android phones carry the "mobile" token in their UA; tablets
		// omit it. This is the one reliable phone/tablet signal Android
		// exposes via User-Agent.
		if strings.Contains(ua, "mobile") {
			return Mobile
		}
		return Tablet
	case containsAny(ua, "ipad", "tablet"):
		return Tablet
	case containsAny(ua, "iphone", "ipod", "mobile"):
		return Mobile
	case containsAny(ua, "windows", "macintosh", "linux", "x11"):
		return Desktop
	default:
		return Unknown
	}
}

// containsAny reports whether s contains at least one of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
