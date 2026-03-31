package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is the on-disk policy file format.
type Config struct {
	Rules Rules  `json:"rules"`
	Log   string `json:"log"`
}

type Rules struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// Loaded is a parsed, ready-to-evaluate policy.
type Loaded struct {
	LogPath    string
	allowRules []parsedRule
	denyRules  []parsedRule
}

type parsedRule struct {
	raw      string
	toolName string
	pattern  string
}

func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "claude-guard", "policy.json")
}

func defaultLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "claude-guard", "decisions.jsonl")
}

// Load reads and parses the policy config.
func Load() (*Loaded, error) {
	path := os.Getenv("CLAUDE_GUARD_CONFIG")
	if path == "" {
		path = defaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	logPath := cfg.Log
	if logPath == "" {
		logPath = defaultLogPath()
	}
	logPath = expandHome(logPath)

	loaded := &Loaded{LogPath: logPath}
	for _, raw := range cfg.Rules.Allow {
		loaded.allowRules = append(loaded.allowRules, parseRule(raw))
	}
	for _, raw := range cfg.Rules.Deny {
		loaded.denyRules = append(loaded.denyRules, parseRule(raw))
	}
	return loaded, nil
}

// Result describes the outcome of a policy evaluation.
type Result struct {
	Decision string // "allow", "deny", or "skip"
	Rule     string // the matching rule (empty for skip)
}

// Evaluate checks deny rules first, then allow rules.
// Deny takes precedence: if both match, the request is denied.
func (l *Loaded) Evaluate(toolName, input string) Result {
	// Deny rules take precedence.
	for _, r := range l.denyRules {
		if r.toolName != toolName {
			continue
		}
		if matchPattern(r.pattern, input) {
			return Result{Decision: "deny", Rule: r.raw}
		}
	}

	// Then check allow rules.
	for _, r := range l.allowRules {
		if r.toolName != toolName {
			continue
		}
		if matchPattern(r.pattern, input) {
			return Result{Decision: "allow", Rule: r.raw}
		}
	}

	return Result{Decision: "skip"}
}

// Match returns the first matching allow rule, or ("", false).
// Deprecated: use Evaluate for deny-aware matching.
func (l *Loaded) Match(toolName, input string) (string, bool) {
	res := l.Evaluate(toolName, input)
	if res.Decision == "allow" {
		return res.Rule, true
	}
	return "", false
}

func parseRule(raw string) parsedRule {
	r := parsedRule{raw: raw}

	idx := strings.Index(raw, "(")
	if idx == -1 {
		// "ToolName" with no parens = match any input for that tool.
		r.toolName = raw
		r.pattern = "*"
		return r
	}

	r.toolName = raw[:idx]
	inner := raw[idx+1:]
	inner = strings.TrimSuffix(inner, ")")
	r.pattern = inner
	return r
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
