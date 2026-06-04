package devicedetector

import _ "embed"

//go:embed brands.json
var brandData []byte

// brandRules holds the ordered, compiled brand rules from brands.json. Short or
// ambiguous tokens are anchored to avoid substring false positives.
var brandRules = mustLoadVersionedRules(brandData)

// detectBrand returns the device brand for userAgent, or "" if none matches.
func detectBrand(userAgent string) string {
	name, _, _ := matchFirst(brandRules, userAgent)
	return name
}
