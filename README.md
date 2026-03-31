# claude-guard

A PermissionRequest hook for [Claude Code](https://claude.ai/code) that auto-approves safe commands based on a policy file and logs every decision for review.

**This is not a tool to bypass your org's security policy.** Org-managed `deny` and `ask` rules are enforced by Claude Code before this hook ever fires - `claude-guard` cannot override them. What it does is complement those guardrails by letting you define a personal allow-list for the safe, routine commands that would otherwise require constant manual approval.

The problem it solves: when org settings set `allowManagedPermissionRulesOnly: true` without any `allow` rules, every Bash command prompts for approval. This leads to decision fatigue - and decision fatigue leads to blindly approving everything, which is worse than having no prompts at all. `claude-guard` lets you be deliberate about what gets auto-approved while logging every decision for audit.

**With great power comes great responsibility.** Every action auto-approved by this tool runs under your identity. You are responsible for your policy - if it's too lax and something goes wrong, your name is on it. Start conservative, review your logs (`claude-guard review`), and broaden rules only as you build confidence. Better yet, ask Claude to run `claude-guard review` and discuss the findings with you - it can help spot rules that are too broad or suggest tighter alternatives.

## How it works

1. Claude Code is about to prompt you for permission
2. `claude-guard` runs as a hook, reads the request from stdin
3. If a **deny** rule matches: blocks the request and logs `deny`
4. If an **allow** rule matches: auto-approves and logs `allow`
5. If no rule matches: does nothing (normal terminal prompt appears) and logs `skip`
6. Stderr shows a one-liner so you see what's happening

Deny rules are evaluated first and take precedence over allow rules. This lets you write broad allow rules with targeted exceptions (e.g. allow `gh api` reads but deny mutating requests).

Managed org `deny` rules still take effect before the hook fires - `claude-guard` can't override org guardrails.

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

Edit to taste:

```json
{
  "rules": {
    "allow": [
      "Bash(gh pr view:*)",
      "Bash(go test:*)",
      "Bash(make:*)",
      "Bash(gh api repos/*/pulls/*/comments:*)"
    ],
    "deny": [
      "Bash(gh api*--input:*)",
      "Bash(gh api*--method DELETE:*)"
    ]
  },
  "log": "~/.local/share/claude-guard/decisions.jsonl"
}
```

### Pattern syntax

- `Bash(prefix:*)` - command starts with `prefix` (token-boundary aware: `git:*` won't match `gitk`)
- `Bash(exact command)` - exact match
- `Bash(*)` - any Bash command (use with caution)
- Works for other tools too: `Edit`, `Write`, `Read` match against `file_path`

### Multi-wildcard patterns

Patterns support multiple `*` wildcards, which goes beyond what Claude Code's native permission system can express. This lets you match commands with variable segments:

```json
"Bash(gh api repos/*/pulls/*/comments:*)"
```

This matches `gh api repos/tigera/matrix/pulls/501/comments --paginate` but not `gh api repos/tigera/matrix/issues/10/comments`.

The `:*` suffix still enforces a token boundary - `repos/*/issues:*` matches `repos/foo/issues --jq .` but not `repos/foo/issues/99/comments`.

### Deny rules

Deny rules use the same pattern syntax and are evaluated before allow rules. This lets you write broad allow rules with targeted exceptions:

```json
{
  "allow": ["Bash(gh api:*)"],
  "deny": [
    "Bash(gh api*--input:*)",
    "Bash(gh api*--method POST:*)",
    "Bash(gh api*--method DELETE:*)"
  ]
}
```

This allows read-only `gh api` calls while blocking mutations. A denied request is logged and blocked - it does not fall through to the terminal prompt.

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
| `Bash(gh api:*)` | Can POST/PUT/DELETE to any GitHub endpoint | Allow with deny rules for `--input`, `--method POST/PUT/DELETE` |
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
