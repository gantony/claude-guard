package policy

import "strings"

// matchPattern checks whether input matches a Claude Code-style permission pattern.
//
// Supported forms:
//
//	"prefix:*"  - input starts with "prefix" (followed by space or end-of-string)
//	"*"         - matches anything
//	"exact"     - input equals "exact"
//	"pre*mid*suf" - glob with one or more * wildcards
//
// Multi-wildcard extends Claude Code's native pattern syntax, which only supports
// a single *. This lets you write rules like "gh api repos/*/pulls/*/comments:*"
// that match across variable path segments.
func matchPattern(pattern, input string) bool {
	if pattern == "*" {
		return true
	}

	// Most common case: "prefix:*" means command starts with prefix.
	// The :* suffix is special - it enforces a token boundary after the prefix.
	if strings.HasSuffix(pattern, ":*") {
		prefix := pattern[:len(pattern)-2]
		// The prefix itself may contain * wildcards, so use glob matching.
		if strings.Contains(prefix, "*") {
			// Check every possible split point where a space (or end) could be the boundary.
			// The prefix must glob-match everything up to a token boundary.
			for i := 0; i <= len(input); i++ {
				if i < len(input) && input[i] != ' ' {
					continue
				}
				if globMatch(prefix, input[:i]) {
					return true
				}
			}
			return false
		}
		if !strings.HasPrefix(input, prefix) {
			return false
		}
		// Ensure prefix matches at a token boundary - "git:*" must not match "gitk".
		if len(input) > len(prefix) && input[len(prefix)] != ' ' {
			return false
		}
		return true
	}

	// Glob matching (supports multiple * wildcards).
	if strings.Contains(pattern, "*") {
		return globMatch(pattern, input)
	}

	// Exact match.
	return input == pattern
}

// globMatch matches input against a pattern with zero or more * wildcards.
// Each * matches any substring (including empty).
func globMatch(pattern, input string) bool {
	// Split pattern by * to get the literal segments between wildcards.
	segments := strings.Split(pattern, "*")

	// Single segment (no wildcards) = exact match.
	if len(segments) == 1 {
		return pattern == input
	}

	// First segment must match at the start.
	if !strings.HasPrefix(input, segments[0]) {
		return false
	}
	pos := len(segments[0])

	// Last segment must match at the end.
	last := segments[len(segments)-1]
	if !strings.HasSuffix(input, last) {
		return false
	}
	end := len(input) - len(last)

	// Middle segments must appear in order between start and end.
	for _, seg := range segments[1 : len(segments)-1] {
		if seg == "" {
			continue
		}
		idx := strings.Index(input[pos:], seg)
		if idx < 0 || pos+idx > end {
			return false
		}
		pos += idx + len(seg)
	}

	return pos <= end
}
