# claude-guard

A PermissionRequest hook for [Claude Code](https://claude.ai/code) that auto-approves safe commands based on a policy file and logs every decision for review.

Built to work alongside org-managed settings that set `allowManagedPermissionRulesOnly: true` - where your personal `allow` rules in `~/.claude/settings.json` are ignored, and every Bash command prompts for approval.

## How it works

1. Claude Code is about to prompt you for permission
2. `claude-guard` runs as a hook, reads the request from stdin
3. If a policy rule matches: auto-approves and logs `allow`
4. If no rule matches: does nothing (normal terminal prompt appears) and logs `skip`
5. Stderr shows a one-liner so you see what's happening

Managed `deny` rules still take effect before the hook fires - `claude-guard` can't override org guardrails.

## Install

```bash
go install github.com/gantony/claude-guard@latest
```

Or build from source:

```bash
git clone https://github.com/gantony/claude-guard.git
cd claude-guard
go build -o claude-guard .
mv claude-guard ~/.local/bin/  # or wherever you keep binaries
```

## Setup

### 1. Create a policy file

```bash
mkdir -p ~/.config/claude-guard
cp policy.example.json ~/.config/claude-guard/policy.json
```

Edit to taste. The format uses the same glob patterns as Claude Code settings:

```json
{
  "rules": {
    "allow": [
      "Bash(gh pr view:*)",
      "Bash(go test:*)",
      "Bash(make:*)"
    ]
  },
  "log": "~/.local/share/claude-guard/decisions.jsonl"
}
```

Pattern syntax:
- `Bash(prefix:*)` - command starts with `prefix` (token-boundary aware: `git:*` won't match `gitk`)
- `Bash(exact command)` - exact match
- `Bash(*)` - any Bash command (use with caution)
- Works for other tools too: `Edit`, `Write`, `Read` match against `file_path`

### 2. Add the hook to your Claude Code settings

Add to `~/.claude/settings.json` under `hooks`:

```json
{
  "hooks": {
    "PermissionRequest": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "claude-guard"
          }
        ]
      }
    ]
  }
}
```

No restart needed - hooks are picked up on next invocation. Policy changes are also picked up immediately (the binary re-reads the config on every call).

## Review decisions

```bash
# Last 24 hours (default)
claude-guard review

# Last 7 days
claude-guard review --since 7d

# Only show unmatched (commands that prompted you)
claude-guard review --unmatched
```

Example output:

```
Decisions (last 24h): 47 allow, 12 skip

Top rules:
  Bash(gh pr view:*)                            15 hits
  Bash(go:*)                                    12 hits
  Bash(make:*)                                  8 hits

Unmatched (prompted in terminal):
  Bash: gh pr create --title ...                (3x)
  Bash: docker push ghcr.io/...                 (2x)

Suggested rules:
  "Bash(gh pr create:*)"
  "Bash(docker push:*)"
```

The feedback loop: review unmatched commands, decide if you're comfortable auto-approving them, add rules to your policy.

## Seed policy notes

The example policy (`policy.example.json`) was seeded from a personal Claude Code settings allow list with these adjustments:

| Original rule | Issue | Seed policy |
|---|---|---|
| `Bash(gh api:*)` | Can POST/PUT/DELETE to any GitHub endpoint | Dropped - let these prompt |
| `Bash(gh run:*)` | Includes `rerun`, `cancel` | Narrowed to `view`, `list` |
| `Bash(gh release:*)` | Includes `create`, `edit` | Narrowed to `view`, `list` |
| `Bash(docker:*)` | Includes `push`, `rm`, `system prune` | Narrowed to `ps`, `images`, `logs` |
| `Bash(kubectl:*)` | Includes `apply`, `delete`, `exec` | Narrowed to `get`, `describe`, `logs` |
| `Bash(helm:*)` | Includes `uninstall`, `upgrade` | Narrowed to `list`, `status`, `get` |
| `Bash(curl:*)` | Can exfiltrate data | Dropped |
| `Bash(xargs:*)` | Executes arbitrary commands | Dropped |

Review your unmatched log and broaden rules as you get comfortable.

## Future work

- **PreToolUse hook**: log all tool calls (including already-allowed ones) for full visibility, not just permission requests
- **Channel integration**: forward permission requests to a centralised web app or Slack for remote approval across multiple sessions
- **Rule suggestions from log**: `claude-guard suggest` to auto-generate policy additions from frequent unmatched patterns
