package repository

import "strings"

// ComputeMatchScore returns a similarity score [0, 1] for material name matching.
// Supports exact match, substring match with ratio bonus, and ordered character overlap.
func ComputeMatchScore(input, dbName, dbSpec string) float64 {
	in := strings.TrimSpace(strings.ToLower(input))
	name := strings.TrimSpace(strings.ToLower(dbName))
	spec := strings.TrimSpace(strings.ToLower(dbSpec))

	full := name
	if spec != "" {
		full = name + " " + spec
	}

	// Exact match — score 1.0
	if in == full || in == name {
		return 1.0
	}

	// Input is a substring of the full name — score 0.80-0.95 based on length ratio
	if strings.Contains(full, in) {
		inLen := len([]rune(in))
		fullLen := len([]rune(full))
		if fullLen > 0 {
			ratio := float64(inLen) / float64(fullLen)
			return 0.80 + (0.15 * ratio)
		}
		return 0.80
	}

	// Full name is a substring of the input (user typed more specifically) — score 0.85
	if strings.Contains(in, name) {
		return 0.85
	}

	// Ordered character overlap (Chinese-friendly fuzzy match) — score up to 0.70
	inRunes := []rune(in)
	nameRunes := []rune(name)

	matchedChars := 0
	lastIdx := 0
	for _, r := range inRunes {
		for j := lastIdx; j < len(nameRunes); j++ {
			if nameRunes[j] == r {
				matchedChars++
				lastIdx = j + 1
				break
			}
		}
	}

	if matchedChars > 0 {
		overlap := float64(matchedChars) / float64(len(inRunes))
		return overlap * 0.70
	}

	return 0.0
}
