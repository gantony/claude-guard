package policy

import "strings"

// matchPattern checks whether input matches a Claude Code-style permission pattern.
//
// Supported forms:
//
//	"prefix:*"  - input starts with "prefix" (followed by space or end-of-string)
//	"*"         - matches anything
//	"exact"     - input equals "exact"
//	"pre*suf"   - simple glob with a single * wildcard
func matchPattern(pattern, input string) bool {
	if pattern == "*" {
		return true
	}

	// Most common case: "prefix:*" means command starts with prefix.
	if strings.HasSuffix(pattern, ":*") {
		prefix := pattern[:len(pattern)-2]
		if !strings.HasPrefix(input, prefix) {
			return false
		}
		// Ensure prefix matches at a token boundary - "git:*" must not match "gitk".
		if len(input) > len(prefix) && input[len(prefix)] != ' ' {
			return false
		}
		return true
	}

	// Simple glob: single * anywhere.
	if star := strings.Index(pattern, "*"); star >= 0 {
		prefix := pattern[:star]
		suffix := pattern[star+1:]
		return strings.HasPrefix(input, prefix) && strings.HasSuffix(input, suffix) && len(input) >= len(prefix)+len(suffix)
	}

	// Exact match.
	return input == pattern
}
