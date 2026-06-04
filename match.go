package devicedetector

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// compileInsensitive validates a non-empty pattern and compiles it
// case-insensitively. Centralizing the "(?i)" flag means individual rules in
// the embedded JSON files cannot silently forget it.
func compileInsensitive(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, fmt.Errorf("empty regex")
	}
	return regexp.Compile("(?i)" + pattern)
}

// versionedRule is a compiled rule that yields a name and a version. The
// version is capture group 1 of the regex (empty when the regex has no group).
type versionedRule struct {
	re   *regexp.Regexp
	name string
}

// versionedRuleJSON is the on-disk shape of a rule in oss.json / browsers.json.
type versionedRuleJSON struct {
	Regex string `json:"regex"`
	Name  string `json:"name"`
}

// loadVersionedRules parses and compiles a versioned ruleset.
func loadVersionedRules(data []byte) ([]versionedRule, error) {
	var raw []versionedRuleJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse rules: %w", err)
	}
	rules := make([]versionedRule, 0, len(raw))
	for i, r := range raw {
		if r.Name == "" {
			return nil, fmt.Errorf("rule at index %d has empty name", i)
		}
		re, err := compileInsensitive(r.Regex)
		if err != nil {
			return nil, fmt.Errorf("bad rule at index %d (%q): %w", i, r.Regex, err)
		}
		rules = append(rules, versionedRule{re: re, name: r.Name})
	}
	return rules, nil
}

// mustLoadVersionedRules is loadVersionedRules for author-controlled embedded
// data: a parse/compile failure is a programmer error, so it panics.
func mustLoadVersionedRules(data []byte) []versionedRule {
	rules, err := loadVersionedRules(data)
	if err != nil {
		panic("devicedetector: " + err.Error())
	}
	return rules
}

// matchFirst returns the name and raw capture group 1 of the first rule that
// matches userAgent (file order, first match wins), or ok=false if none match.
// capture is "" when the regex has no capture group.
func matchFirst(rules []versionedRule, userAgent string) (name, capture string, ok bool) {
	for _, rule := range rules {
		m := rule.re.FindStringSubmatch(userAgent)
		if m == nil {
			continue
		}
		if len(m) > 1 {
			capture = m[1]
		}
		return rule.name, capture, true
	}
	return "", "", false
}

// matchVersioned is matchFirst with the capture treated as a version: "_" is
// normalized to "." (for UA tokens like iOS "17_0"). The version is "" when the
// regex has no capture group or no rule matches.
func matchVersioned(rules []versionedRule, userAgent string) (name, version string, ok bool) {
	name, capture, ok := matchFirst(rules, userAgent)
	return name, strings.ReplaceAll(capture, "_", "."), ok
}
