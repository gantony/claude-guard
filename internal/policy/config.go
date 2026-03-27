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
}

// Loaded is a parsed, ready-to-evaluate policy.
type Loaded struct {
	LogPath string
	rules   []parsedRule
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
		loaded.rules = append(loaded.rules, parseRule(raw))
	}
	return loaded, nil
}

// Match returns the first matching allow rule, or ("", false).
func (l *Loaded) Match(toolName, input string) (string, bool) {
	for _, r := range l.rules {
		if r.toolName != toolName {
			continue
		}
		if matchPattern(r.pattern, input) {
			return r.raw, true
		}
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
