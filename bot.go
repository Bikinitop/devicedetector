package devicedetector

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
)

//go:embed bots.json
var botData []byte

// BotInfo describes a bot identified from a User-Agent string.
type BotInfo struct {
	Name     string
	Category string
}

// botRule is a compiled bot-detection rule.
type botRule struct {
	re       *regexp.Regexp
	name     string
	category string
}

// botRuleJSON is the on-disk shape of a single rule in bots.json.
type botRuleJSON struct {
	Regex    string `json:"regex"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

// botRules holds the ordered, compiled bot rules loaded from bots.json.
// The order in the file is significant: specific bots come first, the
// generic catch-all last, so the most specific rule wins.
var botRules = mustLoadBotRules(botData)

// loadBotRules parses and compiles the embedded bot ruleset.
func loadBotRules(data []byte) ([]botRule, error) {
	var raw []botRuleJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse bots.json: %w", err)
	}
	rules := make([]botRule, 0, len(raw))
	for i, r := range raw {
		re, err := regexp.Compile(r.Regex)
		if err != nil {
			return nil, fmt.Errorf("bad bot regex at index %d (%q): %w", i, r.Regex, err)
		}
		rules = append(rules, botRule{re: re, name: r.Name, category: r.Category})
	}
	return rules, nil
}

// mustLoadBotRules is loadBotRules for author-controlled embedded data:
// a parse/compile failure is a programmer error, so it panics.
func mustLoadBotRules(data []byte) []botRule {
	rules, err := loadBotRules(data)
	if err != nil {
		panic("devicedetector: " + err.Error())
	}
	return rules
}

// detectBot returns the first matching bot rule's info for userAgent, or nil
// if no rule matches. Rules are case-insensitive and matched against the raw
// User-Agent in file order (first match wins).
func detectBot(userAgent string) *BotInfo {
	for _, rule := range botRules {
		if rule.re.MatchString(userAgent) {
			return &BotInfo{Name: rule.name, Category: rule.category}
		}
	}
	return nil
}
