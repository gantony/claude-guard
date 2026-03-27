package review

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gantony/claude-guard/internal/decision"
	"github.com/gantony/claude-guard/internal/policy"
)

func Run(args []string) error {
	cfg, err := policy.Load()
	if err != nil {
		return err
	}

	since := 24 * time.Hour
	unmatchedOnly := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--since":
			if i+1 >= len(args) {
				return fmt.Errorf("--since requires a value (e.g. 24h, 7d, 1w)")
			}
			i++
			d, err := parseDuration(args[i])
			if err != nil {
				return fmt.Errorf("invalid duration %q: %w", args[i], err)
			}
			since = d
		case "--unmatched":
			unmatchedOnly = true
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	cutoff := time.Now().Add(-since)
	entries, err := readLog(cfg.LogPath, cutoff)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Println("No decisions in the specified period.")
		return nil
	}

	allowCount, skipCount := 0, 0
	ruleHits := map[string]int{}
	unmatched := map[string]int{}

	for _, e := range entries {
		switch e.Decision {
		case "allow":
			allowCount++
			ruleHits[e.Rule]++
		case "skip":
			skipCount++
			key := fmt.Sprintf("%s: %s", e.Tool, normalizeCmd(e.Input))
			unmatched[key]++
		}
	}

	fmt.Printf("Decisions (last %s): %d allow, %d skip\n\n", fmtDuration(since), allowCount, skipCount)

	if !unmatchedOnly && len(ruleHits) > 0 {
		fmt.Println("Top rules:")
		for _, kv := range sortedByCount(ruleHits) {
			fmt.Printf("  %-45s %d hits\n", kv.key, kv.count)
		}
		fmt.Println()
	}

	if len(unmatched) > 0 {
		fmt.Println("Unmatched (prompted in terminal):")
		for _, kv := range sortedByCount(unmatched) {
			fmt.Printf("  %-55s (%dx)\n", kv.key, kv.count)
		}
		fmt.Println()

		fmt.Println("Suggested rules:")
		seen := map[string]bool{}
		for _, kv := range sortedByCount(unmatched) {
			parts := strings.SplitN(kv.key, ": ", 2)
			if len(parts) != 2 {
				continue
			}
			rule := suggestRule(parts[0], parts[1])
			if seen[rule] {
				continue
			}
			seen[rule] = true
			fmt.Printf("  %q\n", rule)
		}
	}

	return nil
}

func readLog(path string, after time.Time) ([]decision.Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []decision.Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e decision.Entry
		if json.Unmarshal(scanner.Bytes(), &e) != nil {
			continue
		}
		if e.Timestamp.After(after) {
			entries = append(entries, e)
		}
	}
	return entries, scanner.Err()
}

// normalizeCmd collapses long commands to a prefix for grouping.
func normalizeCmd(input string) string {
	parts := strings.Fields(input)
	if len(parts) <= 4 {
		return input
	}
	return strings.Join(parts[:4], " ") + " ..."
}

// suggestRule infers a prefix rule from an unmatched command.
func suggestRule(tool, cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return fmt.Sprintf("%s(*)", tool)
	}

	// Take the first 1-3 words that look like subcommands (not flags or paths).
	var prefix []string
	for _, p := range parts {
		if strings.HasPrefix(p, "-") || strings.HasPrefix(p, "/") || strings.HasPrefix(p, ".") {
			break
		}
		prefix = append(prefix, p)
		if len(prefix) >= 3 {
			break
		}
	}
	if len(prefix) == 0 {
		prefix = parts[:1]
	}
	return fmt.Sprintf("%s(%s:*)", tool, strings.Join(prefix, " "))
}

type counted struct {
	key   string
	count int
}

func sortedByCount(m map[string]int) []counted {
	out := make([]counted, 0, len(m))
	for k, v := range m {
		out = append(out, counted{k, v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].count > out[j].count })
	return out
}

func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(strings.TrimSuffix(s, "d"), "%d", &days); err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	if strings.HasSuffix(s, "w") {
		var weeks int
		if _, err := fmt.Sscanf(strings.TrimSuffix(s, "w"), "%d", &weeks); err != nil {
			return 0, err
		}
		return time.Duration(weeks) * 7 * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func fmtDuration(d time.Duration) string {
	if d >= 7*24*time.Hour {
		return fmt.Sprintf("%dw", int(d/(7*24*time.Hour)))
	}
	if d >= 24*time.Hour {
		return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
	}
	return d.String()
}
