package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gantony/claude-guard/internal/decision"
	"github.com/gantony/claude-guard/internal/policy"
)

// Input is the JSON that Claude Code sends on stdin for PermissionRequest hooks.
type Input struct {
	SessionID string          `json:"session_id"`
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// Output is the JSON that Claude Code expects on stdout.
type Output struct {
	HookSpecificOutput HookSpecific `json:"hookSpecificOutput"`
}

type HookSpecific struct {
	HookEventName string   `json:"hookEventName"`
	Decision      Decision `json:"decision"`
}

type Decision struct {
	Behavior string `json:"behavior"`
}

func Run() error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var in Input
	if err := json.Unmarshal(data, &in); err != nil {
		return fmt.Errorf("parsing input: %w", err)
	}

	matchStr := extractMatchString(in.ToolName, in.ToolInput)

	cfg, err := policy.Load()
	if err != nil {
		// Can't load policy - log warning and passthrough so the user isn't blocked.
		fmt.Fprintf(os.Stderr, "[claude-guard] SKIP   %s  (config error: %v)\n", truncate(matchStr, 60), err)
		return nil
	}

	res := cfg.Evaluate(in.ToolName, matchStr)

	entry := decision.Entry{
		Timestamp: time.Now().UTC(),
		Tool:      in.ToolName,
		Input:     matchStr,
		Decision:  res.Decision,
		Rule:      res.Rule,
		SessionID: in.SessionID,
	}

	if err := decision.Log(cfg.LogPath, entry); err != nil {
		fmt.Fprintf(os.Stderr, "[claude-guard] warning: log write failed: %v\n", err)
	}

	label := strings.ToUpper(res.Decision)
	switch res.Decision {
	case "allow":
		fmt.Fprintf(os.Stderr, "[claude-guard] %s  %s  (rule: %s)\n", label, truncate(matchStr, 60), res.Rule)
		return json.NewEncoder(os.Stdout).Encode(Output{
			HookSpecificOutput: HookSpecific{
				HookEventName: "PermissionRequest",
				Decision:      Decision{Behavior: "allow"},
			},
		})
	case "deny":
		fmt.Fprintf(os.Stderr, "[claude-guard] %s   %s  (rule: %s)\n", label, truncate(matchStr, 60), res.Rule)
		return json.NewEncoder(os.Stdout).Encode(Output{
			HookSpecificOutput: HookSpecific{
				HookEventName: "PermissionRequest",
				Decision:      Decision{Behavior: "deny"},
			},
		})
	default:
		fmt.Fprintf(os.Stderr, "[claude-guard] SKIP   %s  (no match)\n", truncate(matchStr, 60))
		// Empty stdout = passthrough to normal terminal prompt.
		return nil
	}
}

// extractMatchString pulls the relevant string from tool_input for matching.
func extractMatchString(toolName string, raw json.RawMessage) string {
	switch toolName {
	case "Bash":
		var v struct {
			Command string `json:"command"`
		}
		if json.Unmarshal(raw, &v) == nil && v.Command != "" {
			return v.Command
		}
	case "Edit", "Write", "Read":
		var v struct {
			FilePath string `json:"file_path"`
		}
		if json.Unmarshal(raw, &v) == nil && v.FilePath != "" {
			return v.FilePath
		}
	}
	// Fallback: raw JSON (useful for logging even if no rule matches).
	return string(raw)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
