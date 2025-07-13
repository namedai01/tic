package utils

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"unicode"
)

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

// SanitizeString removes non-printable characters and trims whitespace
func SanitizeString(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, s))
}

// TruncateString truncates a string to the specified length
func TruncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// SliceContains checks if a slice contains a specific string
func SliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ExtractKeywords extracts keywords from text (simple implementation)
func ExtractKeywords(text string, maxKeywords int) []string {
	// Simple keyword extraction - split by spaces and filter
	words := strings.Fields(strings.ToLower(text))
	keywords := make(map[string]bool)
	var result []string

	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
	}

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:")
		if len(word) > 2 && !stopWords[word] && !keywords[word] {
			keywords[word] = true
			result = append(result, word)
			if len(result) >= maxKeywords {
				break
			}
		}
	}

	return result
}
