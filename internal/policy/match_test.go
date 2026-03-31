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

		// Simple glob (single wildcard)
		{"*.go", "main.go", true},
		{"*.go", "main.rs", false},
		{"pre*suf", "pre-middle-suf", true},
		{"pre*suf", "pre-suf", true},
		{"pre*suf", "presuf", true},
		{"pre*suf", "prefix", false},

		// Multi-wildcard glob
		{"gh api repos/*/pulls/*/comments", "gh api repos/tigera/matrix/pulls/501/comments", true},
		{"gh api repos/*/pulls/*/comments", "gh api repos/other/pulls/99/comments", true},
		{"gh api repos/*/pulls/*/comments", "gh api repos/tigera/matrix/issues/10/comments", false},
		{"a*b*c", "abc", true},
		{"a*b*c", "aXbYc", true},
		{"a*b*c", "aXbYcZc", true},
		{"a*b*c", "aXYZ", false},
		{"*mid*", "start-mid-end", true},
		{"*mid*", "mid", true},
		{"*mid*", "nomatch", false},

		// Multi-wildcard with :* suffix (prefix with wildcards in path)
		{"gh api repos/*/pulls/*/comments:*", "gh api repos/tigera/matrix/pulls/501/comments --paginate", true},
		{"gh api repos/*/pulls/*/comments:*", "gh api repos/tigera/matrix/pulls/501/comments --jq .body", true},
		{"gh api repos/*/pulls/*/comments:*", "gh api repos/tigera/matrix/pulls/501/comments", true},
		{"gh api repos/*/pulls/*/comments:*", "gh api repos/tigera/matrix/pulls/501/reviews", false},
		{"gh api repos/*/pulls/*/comments:*", "gh api repos/tigera/matrix/pulls/501/comments/123/replies --input x", false}, // not a token boundary

		// Multi-wildcard :* with token boundary enforcement
		{"gh api repos/*/issues:*", "gh api repos/foo/issues --jq .", true},
		{"gh api repos/*/issues:*", "gh api repos/foo/issues", true},
		{"gh api repos/*/issues:*", "gh api repos/foo/issues/99/comments", false}, // /99 follows without space
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

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		{"abc", "abc", true},
		{"abc", "abd", false},
		{"*", "anything", true},
		{"*", "", true},
		{"a*", "abc", true},
		{"a*", "bcd", false},
		{"*c", "abc", true},
		{"*c", "abd", false},
		{"a*c", "abc", true},
		{"a*c", "aXYZc", true},
		{"a*c", "abd", false},
		{"a*b*c", "abc", true},
		{"a*b*c", "aXbYc", true},
		{"a*b*c", "ac", false},
		{"**", "anything", true},
		{"a**b", "ab", true},
		{"a**b", "aXb", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			got := globMatch(tt.pattern, tt.input)
			if got != tt.want {
				t.Errorf("globMatch(%q, %q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	loaded := &Loaded{
		allowRules: []parsedRule{
			parseRule("Bash(gh api:*)"),
		},
		denyRules: []parsedRule{
			parseRule("Bash(gh api*--input:*)"),
			parseRule("Bash(gh api*--method POST:*)"),
			parseRule("Bash(gh api*--method PUT:*)"),
			parseRule("Bash(gh api*--method DELETE:*)"),
		},
	}

	tests := []struct {
		name    string
		tool    string
		input   string
		wantDec string
	}{
		{"allow read", "Bash", "gh api repos/tigera/matrix/pulls/501/comments --paginate", "allow"},
		{"deny --input", "Bash", "gh api repos/tigera/matrix/pulls/501/comments/123/replies --input /tmp/reply.json", "deny"},
		{"deny --method POST", "Bash", "gh api repos/tigera/matrix/pulls/501/reviews --method POST", "deny"},
		{"deny --method DELETE", "Bash", "gh api repos/tigera/matrix/pulls/501/comments/123 --method DELETE", "deny"},
		{"skip unrelated", "Bash", "curl http://example.com", "skip"},
		{"skip wrong tool", "Write", "/tmp/foo.txt", "skip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := loaded.Evaluate(tt.tool, tt.input)
			if res.Decision != tt.wantDec {
				t.Errorf("Evaluate(%q, %q) = %q, want %q", tt.tool, tt.input, res.Decision, tt.wantDec)
			}
		})
	}
}
