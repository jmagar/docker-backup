package util

import (
	"fmt"
	"path/filepath"
	"strings"
)

// MatchesExclude checks if a given path matches any of the provided glob patterns.
func MatchesExclude(path string, patterns []string) (bool, error) {
	if len(patterns) == 0 {
		return false, nil
	}

	normalizedPath := filepath.ToSlash(path)
	baseName := filepath.Base(normalizedPath)

	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)

		// Specific handling for dir/** patterns to match the directory itself
		if strings.HasSuffix(pattern, "/**") {
			dirPattern := strings.TrimSuffix(pattern, "/**")
			// Match if the path IS the directory OR if the path STARTS WITH the directory followed by a slash
			if normalizedPath == dirPattern || strings.HasPrefix(normalizedPath, dirPattern+"/") {
				return true, nil
			}
			// Fall through for standard match below
		}

		// Specific handling for **/<basename_pattern>
		if strings.HasPrefix(pattern, "**/") {
			basePattern := strings.TrimPrefix(pattern, "**/")
			matchedBase, _ := filepath.Match(basePattern, baseName) // Check base name directly
			if matchedBase {
				return true, nil
			}
			// Fall through for standard match below
		}

		// Standard match against the full relative path
		matched, err := filepath.Match(pattern, normalizedPath)
		if err != nil {
			return false, fmt.Errorf("invalid exclude pattern '%s': %w", pattern, err)
		}
		if matched {
			return true, nil
		}

		// --- Added Check: Explicitly check for directory name within path for wildcard patterns ---
		// This helps catch cases like `*/cache/*` matching `appdata/service/cache/file`
		if strings.Contains(pattern, "/*/") {
			parts := strings.Split(pattern, "/*/")
			if len(parts) == 2 && parts[0] == "*" && parts[1] == "*" {
				dirToMatch := strings.TrimSuffix(strings.TrimPrefix(pattern, "*/"), "/*")
				if strings.Contains(normalizedPath, "/"+dirToMatch+"/") {
					return true, nil
				}
			}
		}
		// --- End Added Check ---
	}

	return false, nil
}
