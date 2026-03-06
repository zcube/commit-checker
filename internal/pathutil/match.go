package pathutil

import "path/filepath"

// MatchesAny reports whether path matches any of the given glob patterns.
// Patterns are matched against the path using filepath.Match semantics.
// "**" glob matching is supported by also testing the base name alone.
func MatchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		// Try full path match
		if m, _ := filepath.Match(pattern, path); m {
			return true
		}
		// Try matching just the base name
		if m, _ := filepath.Match(pattern, filepath.Base(path)); m {
			return true
		}
		// Try matching with forward-slash normalisation
		if matchDoubleStarGlob(path, pattern) {
			return true
		}
	}
	return false
}

// matchDoubleStarGlob handles patterns containing "**" by splitting on "/" and
// matching path segments progressively. It supports patterns like "vendor/**"
// and "**/generated/*.go".
func matchDoubleStarGlob(path, pattern string) bool {
	// Use filepath.ToSlash for consistent separator handling
	pathParts := splitPath(filepath.ToSlash(path))
	patParts := splitPath(filepath.ToSlash(pattern))
	return matchParts(pathParts, patParts)
}

func splitPath(p string) []string {
	var parts []string
	for _, seg := range filepath.SplitList(p) {
		if seg != "" {
			parts = append(parts, seg)
		}
	}
	// filepath.SplitList splits on OS list separator; use manual split instead
	_ = parts
	result := []string{}
	current := ""
	for _, ch := range p {
		if ch == '/' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func matchParts(pathParts, patParts []string) bool {
	if len(patParts) == 0 {
		return len(pathParts) == 0
	}
	if patParts[0] == "**" {
		// ** can match zero or more path segments
		for i := 0; i <= len(pathParts); i++ {
			if matchParts(pathParts[i:], patParts[1:]) {
				return true
			}
		}
		return false
	}
	if len(pathParts) == 0 {
		return false
	}
	m, _ := filepath.Match(patParts[0], pathParts[0])
	if !m {
		return false
	}
	return matchParts(pathParts[1:], patParts[1:])
}
