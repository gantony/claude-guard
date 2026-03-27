package policy

import "testing"

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		// Wildcard
		{"*", "anything", true},
		{"*", "", true},

		// Prefix matching
		{"git:*", "git status", true},
		{"git:*", "git", true},
		{"git:*", "gitk", false}, // must match at token boundary
		{"gh pr view:*", "gh pr view 123", true},
		{"gh pr view:*", "gh pr view", true},
		{"gh pr view:*", "gh pr viewpoint", false},
		{"make:*", "make ci", true},
		{"./bin/golangci-lint:*", "./bin/golangci-lint run", true},

		// Exact match
		{"./bin/tagctl --help", "./bin/tagctl --help", true},
		{"./bin/tagctl --help", "./bin/tagctl --version", false},

		// Simple glob
		{"*.go", "main.go", true},
		{"*.go", "main.rs", false},
		{"pre*suf", "pre-middle-suf", true},
		{"pre*suf", "pre-suf", true},
		{"pre*suf", "presuf", true},
		{"pre*suf", "prefix", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			got := matchPattern(tt.pattern, tt.input)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}
