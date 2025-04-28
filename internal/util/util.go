package util

import (
	"fmt"
	"path/filepath"
)

// MatchesExclude checks if a given path matches any of the provided glob patterns.
func MatchesExclude(path string, patterns []string) (bool, error) {
	if len(patterns) == 0 {
		return false, nil // No patterns means no match
	}

	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, path)
		if err != nil {
			// Pattern syntax error - this should ideally be validated earlier,
			// but we return the error here if encountered.
			return false, fmt.Errorf("invalid exclude pattern '%s': %w", pattern, err)
		}
		if matched {
			return true, nil // Found a match
		}
	}

	return false, nil // No patterns matched
}
