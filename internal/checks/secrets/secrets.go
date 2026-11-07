// Package secrets is the Check no-secrets-in-commits, and the kinds of Secret it knows.
package secrets

import (
	"hash/crc32"
	"regexp"
	"strconv"
	"strings"

	"github.com/bunderlog/hathi/internal/checks"
)

// Allow marks a false alarm: on a line, or on an Env file's first line. `baloo`'s mark passes too,
// so lines already marked still pass, and gitleaks', so one mark serves both (ADR secrets).
var Allow = []string{"hathi:allow-secret", "baloo:allow-secret", "gitleaks:allow"}

// Each kind of Secret, by its name; the first that matches names it, so the more specific come
// first. Each has a prefix of its own (ADR secrets); a private key, found by its body, is apart.
var kinds = []struct {
	what    string
	pattern *regexp.Regexp
}{
	{"an AWS access key", regexp.MustCompile(`\b(?:AKIA|ASIA|ABIA|ACCA)[A-Z2-7]{16}\b`)},
	{"a GitHub token", regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{60,})`)},
	{"a GitLab token", regexp.MustCompile(
		`\bgl(?:pat|dt|rt|cbt|ptt|ft|imt|agent|oas|soat|ffct)-[0-9A-Za-z_.-]{20,}`)},
	{"a Slack webhook URL", regexp.MustCompile(
		`\bhooks\.slack\.com/(?:services|workflows|triggers)/[A-Za-z0-9+/]{43,56}`)},
	{"a Slack token", regexp.MustCompile(`\b(?:xox[abeoprs]|xapp-\d)-[A-Za-z0-9-]{10,}`)},
	{"a Stripe live key", regexp.MustCompile(`\b[rs]k_live_[A-Za-z0-9]{20,}`)},
	{"an Anthropic API key", regexp.MustCompile(`\bsk-ant-[a-z]+\d{2}-[A-Za-z0-9_-]{40,}`)},
	// OpenAI's keys hold T3BlbkFJ, "OpenAI" in base64.
	{"an OpenAI API key", regexp.MustCompile(
		`\bsk-(?:(?:proj|svcacct|admin)-)?[A-Za-z0-9_-]{20,}T3BlbkFJ[A-Za-z0-9_-]{20,}`)},
	{"a Google API key", regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`)},
	{"an npm token", regexp.MustCompile(`\bnpm_[A-Za-z0-9]{36}\b`)},
	{"a PyPI token", regexp.MustCompile(`\bpypi-AgEIcHlwaS5vcmc[A-Za-z0-9_-]{50,}`)},
	{"a Docker Hub token", regexp.MustCompile(`\bdckr_(?:pat|oat)_[A-Za-z0-9_-]{20,}`)},
	{"a Hugging Face token", regexp.MustCompile(`\b(?:hf_|api_org_)[A-Za-z]{34}\b`)},
	{"a SendGrid API key", regexp.MustCompile(`\bSG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}\b`)},
	{"a DigitalOcean token", regexp.MustCompile(`\bdo[opr]_v1_[a-f0-9]{64}\b`)},
	{"a Shopify token", regexp.MustCompile(`\bshp(?:at|ca|pa|ss)_[a-fA-F0-9]{32}\b`)},
	{"a Vault token", regexp.MustCompile(`\bhv[sb]\.[A-Za-z0-9_-]{90,}`)},
	{"an age secret key", regexp.MustCompile(
		`\bAGE-SECRET-KEY-1[QPZRY9X8GF2TVDW0S3JN54KHCE6MUA7L]{58}\b`)},
	{"a Telegram bot token", regexp.MustCompile(`\b[0-9]{5,16}:A[0-9A-Za-z_-]{34}\b`)},
	{"a JWT", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{20,}`)},
	// The password in a URL, between `://user:` and `@`, unless it is a placeholder: `$VAR`,
	// `${VAR}`, `<password>`, `****`. Whitespace is ASCII's, \v too, as in `baloo`.
	{"a password in a URL", regexp.MustCompile(
		`\b[A-Za-z][A-Za-z0-9+.-]*://[^\t-\r :/@]*:[^\t-\r /@$<{*][^\t-\r /@]*@`)},
}

// A match holding one of these is a placeholder, not a Secret, in any case (ADR secrets).
var placeholders = []string{"example", "sample", "xxxxxx", "aaaaaa", "abcde", "00000", "*****"}

var (
	privateKey = regexp.MustCompile(`-----BEGIN (?:[A-Z]+ )*PRIVATE KEY-----`)
	// A run of base64 as a private key's body has it, and a PEM header line before the body.
	base64    = regexp.MustCompile(`[A-Za-z0-9+/]{40,}`)
	pemHeader = regexp.MustCompile(`^[A-Za-z-]+: `)
)

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
	// The line of a private key's BEGIN whose body may follow on the next added lines; 0 for none.
	key := 0
	for s := range strings.SplitSeq(text, "\n") {
		switch {
		case strings.HasPrefix(s, "diff --git "):
			header, key = true, 0
		case header && strings.HasPrefix(s, "+++ "):
			path = name(strings.TrimPrefix(s, "+++ "))
		case strings.HasPrefix(s, "@@"):
			header, key = false, 0
			if m := hunk.FindStringSubmatch(s); m != nil {
				line, _ = strconv.Atoi(m[1])
			}
		case !header && strings.HasPrefix(s, "+"):
			added := s[1:]
			if key > 0 {
				switch {
				case isBody(added):
					found = append(found, checks.Finding{File: path, Line: key, Reason: "a private key"})
					key = 0
				case added != "" && !pemHeader.MatchString(added):
					key = 0
				}
			}
			if !allowed(s) {
				if at := privateKey.FindStringIndex(added); at != nil {
					if isBody(added[at[1]:]) {
						found = append(found, checks.Finding{File: path, Line: line, Reason: "a private key"})
					} else {
						key = line
					}
				} else if what := secret(added); what != "" {
					found = append(found, checks.Finding{File: path, Line: line, Reason: what})
				}
			}
			line++
		}
	}
	return found
}

// secret returns the kind of the first Secret in `line` that isn't a placeholder, or "".
func secret(line string) string {
	for _, k := range kinds {
		for _, m := range k.pattern.FindAllString(line, -1) {
			if !isPlaceholder(m) && checksumHolds(m) {
				return k.what
			}
		}
	}
	return ""
}

func isPlaceholder(match string) bool {
	match = strings.ToLower(match)
	for _, p := range placeholders {
		if strings.Contains(match, p) {
			return true
		}
	}
	return false
}

// base62 is the alphabet of a GitHub token's checksum.
const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// checksumHolds reports whether a classic GitHub token ends in the checksum of its 30 random
// characters: their CRC32 in base62, padded to 6 with 0 (ADR secrets). A match of another kind
// holds.
func checksumHolds(match string) bool {
	if !strings.HasPrefix(match, "ghp_") || len(match) != 40 {
		return true
	}
	sum := crc32.ChecksumIEEE([]byte(match[4:34]))
	want := []byte("000000")
	for i := 5; sum > 0; i-- {
		want[i] = base62[sum%62]
		sum /= 62
	}
	return match[34:] == string(want)
}

// isBody reports whether `text` holds a private key's body: a run of base64 with a digit, which a
// long name in code hardly has.
func isBody(text string) bool {
	for _, run := range base64.FindAllString(text, -1) {
		if strings.ContainsAny(run, "0123456789") {
			return true
		}
	}
	return false
}
