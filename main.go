package main

import (
	"fmt"
	"os"

	"github.com/gantony/claude-guard/internal/hook"
	"github.com/gantony/claude-guard/internal/review"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "review":
			if err := review.Run(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "claude-guard review: %v\n", err)
				os.Exit(1)
			}
			return
		case "version", "--version":
			fmt.Printf("claude-guard %s\n", version)
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	// Default: hook mode (reads stdin, evaluates policy, writes decision to stdout)
	if err := hook.Run(); err != nil {
		// Errors go to stderr; empty stdout = passthrough to terminal prompt.
		// This ensures claude-guard issues never block the user's workflow.
		fmt.Fprintf(os.Stderr, "[claude-guard] error: %v\n", err)
	}
}

func printUsage() {
	fmt.Print(`Usage: claude-guard [command]

Commands:
  (default)   Run as a Claude Code PermissionRequest hook (reads stdin)
  review      Summarise recent decisions from the log
  version     Print version
  help        Show this help

Review flags:
  --since <duration>   Time window: 24h, 7d, 1w (default: 24h)
  --unmatched          Show only unmatched (prompted) decisions

Config: ~/.config/claude-guard/policy.json  (override with CLAUDE_GUARD_CONFIG)
Log:    ~/.local/share/claude-guard/decisions.jsonl
`)
}
