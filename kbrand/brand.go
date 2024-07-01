package kbrand

import (
	"regexp"
	"strings"
	"unicode"
)

// Brand represents a brand with its raw name, English part, and Chinese part.
type Brand struct {
	Raw string
	EN  string
	CN  string
}

// containsChinese checks if a string contains Chinese characters.
func containsChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// containsEnglish checks if a string contains English characters.
func containsEnglish(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Latin, r) {
			return true
		}
	}
	return false
}

// ParseBrand parses a raw brand name into a Brand structure.
func ParseBrand(raw string) Brand {
	brand := Brand{Raw: raw}

	// Regular expressions to match Chinese and English parts
	reChinese := regexp.MustCompile(`[\p{Han}]+`)
	//
	reEnglish := regexp.MustCompile(`[a-zA-Z0-9]+`)
	// reEnglish := regexp.MustCompile(`[a-zA-Z]+`)

	// Find all Chinese parts
	chineseParts := reChinese.FindAllString(raw, -1)
	// Find all English parts
	englishParts := reEnglish.FindAllString(raw, -1)

	// Combine Chinese and English parts
	brand.CN = strings.Join(chineseParts, " ")
	brand.EN = strings.Join(englishParts, " ")

	// Special handling for cases like "南极人（Nanjiren）"
	if strings.Contains(raw, "（") && strings.Contains(raw, "）") {
		leftBracketIndex := strings.Index(raw, "（")
		rightBracketIndex := strings.Index(raw, "）")
		innerText := raw[leftBracketIndex+3 : rightBracketIndex]

		if containsChinese(innerText) {
			brand.CN = innerText
			brand.EN = raw[:leftBracketIndex]
		} else {
			brand.EN = innerText
			brand.CN = raw[:leftBracketIndex]
		}
	}

	// Trim any leading or trailing whitespace
	brand.EN = strings.TrimSpace(brand.EN)
	brand.CN = strings.TrimSpace(brand.CN)

	return brand
}
