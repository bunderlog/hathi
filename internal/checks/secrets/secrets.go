// Package secrets is the Check no-secrets-in-commits, and the kinds of Secret it knows.
package secrets

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/bunderlog/hathi/internal/checks"
)

// Allow marks a false alarm: on a line, or on an Env file's first line. `baloo`'s mark passes too,
// so lines already marked still pass, and gitleaks', so one mark serves both (ADR secrets).
var Allow = []string{"hathi:allow-secret", "baloo:allow-secret", "gitleaks:allow"}

// Each kind of Secret, by its name; the first that matches names it, so the more specific come
// first.
var kinds = []struct {
	what    string
	pattern *regexp.Regexp
}{
	{"a private key", regexp.MustCompile(`-----BEGIN (?:[A-Z]+ )*PRIVATE KEY-----`)},
	{"an AWS access key", regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)},
	{"a GitHub token", regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{60,})`)},
	{"a Slack token", regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}`)},
	{"a Stripe live key", regexp.MustCompile(`\b[rs]k_live_[A-Za-z0-9]{20,}`)},
	{"an Anthropic API key", regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}`)},
	{"an OpenAI API key", regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_-]{20,}`)},
	{"a Google API key", regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`)},
	{"a JWT", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{20,}`)},
	// The password in a URL, between `://user:` and `@`, unless it is a placeholder: `$VAR`,
	// `${VAR}`, `<password>`, `****`. Whitespace is ASCII's, \v too, as in `baloo`.
	{"a password in a URL", regexp.MustCompile(
		`\b[A-Za-z][A-Za-z0-9+.-]*://[^\t-\r :/@]*:[^\t-\r /@$<{*][^\t-\r /@]*@`)},
}

var (
	// An Env file: .env, .env.local, prod.env, .envrc…
	envFile = regexp.MustCompile(`(^|/)(\.env(\.[^/]*)?|\.envrc|[^/]*\.env)$`)
	// An Env file's template, meant to be committed.
	template = regexp.MustCompile(`\.(example|sample|template|dist)$`)
	hunk     = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)`)
)

// The diff of the files added, copied, changed or renamed, as Git itself prints it, whatever the
// repo's config says of diffs.
var diff = []string{"--no-ext-diff", "--no-textconv", "--no-color", "--diff-filter=ACMR"}

// Check is no-secrets-in-commits: each Env file the changes add or change, and each Secret on a
// line they add, by its line in the file they leave, never the Secret itself. A line marked Allow
// passes, and an Env file marked on its first line, whose lines are still looked through.
var Check = checks.Check{
	Name: "no-secrets-in-commits",
	Hook: "pre-commit",
	Find: func(in checks.Input) ([]checks.Finding, error) {
		names, err := in.Changes.Diff(nil, append(diff, "--name-only", "-z")...)
		if err != nil {
			return nil, err
		}
		var found []checks.Finding
		for path := range strings.SplitSeq(strings.TrimSuffix(names, "\x00"), "\x00") {
			if !IsEnvFile(path) {
				continue
			}
			text, err := in.Changes.File(path)
			if err != nil {
				return nil, err
			}
			if first, _, _ := strings.Cut(text, "\n"); !allowed(first) {
				found = append(found, checks.Finding{
					File:   path,
					Reason: "an Env file; keep it out of Git, or put " + Allow[0] + " on its first line",
				})
			}
		}
		added, err := in.Changes.Diff(nil, append(diff, "--no-prefix", "-U0")...)
		if err != nil {
			return nil, err
		}
		return append(found, scan(added)...), nil
	},
}

// IsEnvFile reports whether the file at `path` is an Env file, not a template of one.
func IsEnvFile(path string) bool {
	return envFile.MatchString(path) && !template.MatchString(path)
}

func allowed(line string) bool {
	for _, mark := range Allow {
		if strings.Contains(line, mark) {
			return true
		}
	}
	return false
}

// name is the path a diff's `+++ ` line names: Git puts a tab after one with a space, and quotes
// one with a quote, a backslash or a control character, as C does.
func name(path string) string {
	path = strings.TrimSuffix(path, "\t")
	if unquoted, err := strconv.Unquote(path); err == nil && strings.HasPrefix(path, `"`) {
		return unquoted
	}
	return path
}

// scan returns each Secret on a line the diff `text` adds.
func scan(text string) []checks.Finding {
	var found []checks.Finding
	path, line, header := "", 0, false
	for s := range strings.SplitSeq(text, "\n") {
		switch {
		case strings.HasPrefix(s, "diff --git "):
			header = true
		case header && strings.HasPrefix(s, "+++ "):
			path = name(strings.TrimPrefix(s, "+++ "))
		case strings.HasPrefix(s, "@@"):
			header = false
			if m := hunk.FindStringSubmatch(s); m != nil {
				line, _ = strconv.Atoi(m[1])
			}
		case !header && strings.HasPrefix(s, "+"):
			if !allowed(s) {
				for _, k := range kinds {
					if k.pattern.MatchString(s) {
						found = append(found, checks.Finding{File: path, Line: line, Reason: k.what})
						break
					}
				}
			}
			line++
		}
	}
	return found
}
