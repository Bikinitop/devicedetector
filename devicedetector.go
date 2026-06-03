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
}

// Detector parses User-Agent strings. Create one with New and reuse it;
// it is safe for concurrent use because it holds no mutable state.
type Detector struct{}

// New returns a ready-to-use Detector.
func New() *Detector {
	return &Detector{}
}

// Detect inspects a User-Agent string and returns the Device it describes.
func (d *Detector) Detect(userAgent string) Device {
	return Device{
		Type:      classify(strings.ToLower(userAgent)),
		UserAgent: userAgent,
	}
}

// classify maps a lower-cased User-Agent string to a DeviceType.
//
// TODO(you): implement the classification rules. See the note from Claude
// in the conversation — this is the core heuristic and the design choices
// here (rule ordering, which tokens to match) shape the whole library.
func classify(ua string) DeviceType {
	// Replace this stub with real detection logic.
	return Unknown
}
