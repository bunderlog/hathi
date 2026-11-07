// Package secrets is the Check no-secrets-in-commits, and the kinds of Secret it knows.
package secrets

import (
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"regexp"
	"strconv"
	"strings"

	"github.com/bunderlog/hathi/internal/checks"
)

// Allow marks a false alarm: on a line, or on an Env file's first line. `baloo`'s mark passes too,
// so lines already marked still pass, and gitleaks', so one Allow mark serves both (ADR secrets).
var Allow = []string{"hathi:allow-secret", "baloo:allow-secret", "gitleaks:allow"}

// Each kind of Secret, by its name; the first that matches names it, so the more specific come
// first. Each has a prefix of its own (ADR secrets); a private key, found by its body, is apart.
var kinds = []struct {
	what    string
	pattern *regexp.Regexp
}{
	{"an AWS access key", regexp.MustCompile(`\b(?:AKIA|ASIA|ABIA|ACCA)[A-Z2-7]{16}\b`)},
	// A classic ghp_ token, 36 characters, is one of the checksummed kinds.
	{"a GitHub token", regexp.MustCompile(
		`\b(?:gh[ousr]_[A-Za-z0-9]{36,}|ghp_[A-Za-z0-9]{37,}|github_pat_[A-Za-z0-9_]{60,})`)},
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
	{"an OpenRouter API key", regexp.MustCompile(`\bsk-or-v1-[0-9a-f]{64}\b`)},
	{"a Groq API key", regexp.MustCompile(`\bgsk_[A-Za-z0-9]{52}\b`)},
	{"an xAI API key", regexp.MustCompile(`\bxai-[0-9A-Za-z_]{80}\b`)},
	{"a Replicate token", regexp.MustCompile(`\br8_[0-9A-Za-z_-]{37}`)},
	{"a Perplexity API key", regexp.MustCompile(`\bpplx-[A-Za-z0-9]{48}\b`)},
	{"a LangSmith API key", regexp.MustCompile(`\blsv2_(?:pt|sk)_[a-f0-9]{32}_[a-f0-9]{10}\b`)},
	{"a Pinecone API key", regexp.MustCompile(`\bpcsk_[A-Za-z0-9]{5,6}_[A-Za-z0-9]{63}\b`)},
	{"a Google API key", regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`)},
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
	{"a webhook signing secret", regexp.MustCompile(`\bwhsec_[A-Za-z0-9+/=]{32,}`)},
	{"a Supabase token", regexp.MustCompile(`\bsbp_[a-f0-9]{40}\b`)},
	{"a Sentry token", regexp.MustCompile(
		`\b(?:sntryu_[a-f0-9]{64}\b|sntrys_eyJ[A-Za-z0-9+/=_]{50,})`)},
	{"a Fly.io token", regexp.MustCompile(`\b(?:fo1_[\w-]{43}|fm[12][ar]?_[A-Za-z0-9+/]{100,})`)},
	{"a Linear API key", regexp.MustCompile(`\blin_api_[A-Za-z0-9]{40}\b`)},
	{"a Notion token", regexp.MustCompile(`\bntn_[0-9]{11}[A-Za-z0-9]{35}\b`)},
	{"a Figma token", regexp.MustCompile(`\bfigd_[A-Za-z0-9_-]{40}`)},
	{"a Doppler token", regexp.MustCompile(
		`\bdp\.(?:ct|pt|st(?:\.[a-z0-9_-]{2,35})?|sa|scim|audit)\.[A-Za-z0-9]{40,44}\b`)},
	{"a 1Password token", regexp.MustCompile(`\bops_eyJ[A-Za-z0-9+/]{250,}`)},
	{"a Heroku API key", regexp.MustCompile(`\bHRKU-AA[0-9A-Za-z_-]{58}`)},
	{"a Terraform Cloud token", regexp.MustCompile(`\b[a-z0-9]{14}\.atlasv1\.[a-z0-9_=-]{60,70}`)},
	{"a Pulumi token", regexp.MustCompile(`\bpul-[a-f0-9]{40}\b`)},
	{"a Postman API key", regexp.MustCompile(`\bPMAK-[a-f0-9]{24}-[a-f0-9]{34}\b`)},
	{"a New Relic key", regexp.MustCompile(`\bNR(?:AK|II)-[A-Za-z0-9_-]{27,32}`)},
	{"a Databricks token", regexp.MustCompile(`\bdapi[a-f0-9]{32}(?:-\d)?\b`)},
	{"a Discord webhook URL", regexp.MustCompile(
		`\bdiscord(?:app)?\.com/api/webhooks/[0-9]{17,20}/[A-Za-z0-9_-]{60,68}`)},
	{"a Microsoft Teams webhook URL", regexp.MustCompile(
		`\.webhook\.office\.com/webhookb2/[a-f0-9-]{36}@[a-f0-9-]{36}/IncomingWebhook/` +
			`[a-f0-9]{32}/[a-f0-9-]{36}`)},
	{"a Grafana token", regexp.MustCompile(`\b(?:glsa_[A-Za-z0-9]{32}_[A-Fa-f0-9]{8}\b|` +
		`glc_[A-Za-z0-9+/]{32,400}|eyJrIjoi[A-Za-z0-9]{70,400})`)},
	{"a Tailscale key", regexp.MustCompile(`\btskey-[a-z]+-[0-9A-Za-z_]+-[0-9A-Za-z_]{20,}`)},
	{"a CircleCI token", regexp.MustCompile(`\bCCIPAT_[A-Za-z0-9]{22}_[a-fA-F0-9]{40}\b`)},
	{"a Buildkite token", regexp.MustCompile(`\bbkua_[a-z0-9]{40}\b`)},
	{"a Docker Swarm token", regexp.MustCompile(
		`\b(?:SWMTKN-1-[0-9a-z]{50}-[0-9a-z]{25}\b|SWMKEY-1-[A-Za-z0-9+/]{43})`)},
	{"a Dynatrace token", regexp.MustCompile(`\bdt0c01\.[A-Za-z0-9]{24}\.[A-Za-z0-9]{64}\b`)},
	{"a RubyGems key", regexp.MustCompile(`\brubygems_[a-f0-9]{48}\b`)},
	{"a Clojars token", regexp.MustCompile(`\bCLOJARS_[a-z0-9]{60}\b`)},
	{"a PostHog personal API key", regexp.MustCompile(`\bphx_[A-Za-z0-9_]{43,48}\b`)},
	{"a Sourcegraph token", regexp.MustCompile(`\bsgp_(?:[a-fA-F0-9]{16}_|local_)?[a-fA-F0-9]{40}\b`)},
	{"a Mapbox secret token", regexp.MustCompile(`\bsk\.eyJ[A-Za-z0-9._-]{80,240}`)},
	{"a PlanetScale token", regexp.MustCompile(`\bpscale_(?:tkn|oauth|pw)_[\w=.-]{32,64}`)},
	{"a Prefect key", regexp.MustCompile(`\bpnu_[A-Za-z0-9]{36}\b`)},
	{"a ReadMe API key", regexp.MustCompile(`\brdme_[a-z0-9]{70}\b`)},
	{"a Contentful token", regexp.MustCompile(`\bCFPAT-[A-Za-z0-9_-]{43}`)},
	{"a Typeform token", regexp.MustCompile(`\btfp_[A-Za-z0-9_]{40,59}\b`)},
	// Resend's keys are base58: no 0, O, I or l.
	{"a Resend API key", regexp.MustCompile(
		`\bre_[1-9A-HJ-NP-Za-km-z]{8}_[1-9A-HJ-NP-Za-km-z]{24}\b`)},
	{"a Shippo live token", regexp.MustCompile(`\bshippo_live_[a-fA-F0-9]{40}\b`)},
	{"a JWT", regexp.MustCompile(
		`\beyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{20,}`)},
	// The password in a URL, between `://user:` and `@`, unless it is a placeholder: `$VAR`,
	// `${VAR}`, `<password>`, `****`. Whitespace is ASCII's, \v too, as in `baloo`.
	{"a password in a URL", regexp.MustCompile(
		`\b[A-Za-z][A-Za-z0-9+.-]*://[^\t-\r :/@]*:[^\t-\r /@$<{*][^\t-\r /@]*@`)},
}

// The kinds whose last characters are a checksum of the rest, each with the check of its own (ADR
// secrets); one whose checksum fails is a Placeholder, and passes.
var checksummed = []struct {
	what    string
	pattern *regexp.Regexp
	holds   func(match string) bool
}{
	{"a GitHub token", regexp.MustCompile(`\bghp_[A-Za-z0-9]{36}\b`), base62Checksum},
	{"an npm token", regexp.MustCompile(`\bnpm_[A-Za-z0-9]{36}\b`), base62Checksum},
	// The CRC32 of the 40 characters after the prefix, in lower-case hex.
	{"a Cloudflare token", regexp.MustCompile(`\bcf(?:k|ut|at)_[A-Za-z0-9]{40}[a-f0-9]{8}\b`),
		func(m string) bool {
			return m[len(m)-8:] == fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(m[len(m)-48:len(m)-8])))
		}},
	// The CRC32 of all 184 characters before it, in upper-case hex.
	{"an Atlassian API token", regexp.MustCompile(`\bATATT3[A-Za-z0-9_=-]{178}[A-F0-9]{8}\b`),
		func(m string) bool {
			return m[184:] == fmt.Sprintf("%08X", crc32.ChecksumIEEE([]byte(m[:184])))
		}},
}

// A match holding one of these is a placeholder, not a Secret, in any case (ADR secrets).
var placeholders = []string{"example", "sample", "xxxxxx", "aaaaaa", "abcde", "00000", "*****"}

var (
	privateKey = regexp.MustCompile(`-----BEGIN (?:[A-Z]+ )*PRIVATE KEY(?: BLOCK)?-----`)
	// A PEM file in base64, as a kubeconfig's client-key-data holds one: "-----BEGIN" encoded.
	encodedPEM = regexp.MustCompile(`LS0tLS1CRUdJTi[A-Za-z0-9+/]{20,}`)
	// A run of base64 as a private key's body has it, and a PEM header line before the body.
	base64Run = regexp.MustCompile(`[A-Za-z0-9+/]{40,}`)
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
	Name:    "no-secrets-in-commits",
	Hook:    "pre-commit",
	Explain: explanation,
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
				} else if hasEncodedKey(added) {
					found = append(found, checks.Finding{File: path, Line: line, Reason: "a private key"})
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
	for _, k := range checksummed {
		for _, m := range k.pattern.FindAllString(line, -1) {
			if !isPlaceholder(m) && k.holds(m) {
				return k.what
			}
		}
	}
	for _, k := range kinds {
		for _, m := range k.pattern.FindAllString(line, -1) {
			if !isPlaceholder(m) {
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

// base62 is the alphabet of GitHub's and npm's checksum.
const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// base62Checksum reports whether a GitHub or npm token, its prefix and 36 characters, ends in the
// CRC32 of its 30 random characters in base62, padded to 6 with 0.
func base62Checksum(match string) bool {
	random, sum := match[len(match)-36:len(match)-6], match[len(match)-6:]
	checksum := crc32.ChecksumIEEE([]byte(random))
	want := []byte("000000")
	for i := 5; checksum > 0; i-- {
		want[i] = base62[checksum%62]
		checksum /= 62
	}
	return sum == string(want)
}

// hasEncodedKey reports whether `line` holds a PEM private key in base64. Not one of the kinds, as
// its base64 holds runs such as AAAAAA that would make it a placeholder.
func hasEncodedKey(line string) bool {
	for _, run := range encodedPEM.FindAllString(line, -1) {
		decoded, _ := base64.StdEncoding.DecodeString(run[:len(run)/4*4])
		if strings.Contains(string(decoded), "PRIVATE KEY") {
			return true
		}
	}
	return false
}

// isBody reports whether `text` holds a private key's body: a run of base64 with a digit, which a
// long name in code hardly has.
func isBody(text string) bool {
	for _, run := range base64Run.FindAllString(text, -1) {
		if strings.ContainsAny(run, "0123456789") {
			return true
		}
	}
	return false
}

// explanation is what `hathi explain` prints for the Check: what it looks for, how to fix a
// failure, and how to mark a false alarm.
const explanation = `no-secrets-in-commits (pre-commit) looks for a Secret on a line the
changes add and for an Env file they add or change, such as .env, .env.local or .envrc; a
template of one (.example, .sample, .template, .dist) passes. A Secret is a private key with its
body, in PEM, PGP or base64; a key or token with a prefix of its own from a cloud (AWS, Google,
Cloudflare, DigitalOcean, Fly.io, Heroku, Supabase, PlanetScale, Databricks, Vault, Doppler,
1Password, Tailscale, Terraform Cloud, Pulumi), a code host, registry or CI (GitHub, GitLab,
Atlassian, npm, PyPI, RubyGems, Clojars, Docker Hub, Docker Swarm, CircleCI, Buildkite, Hugging
Face, Sourcegraph), an AI provider (Anthropic, OpenAI, OpenRouter, Groq, xAI, Replicate,
Perplexity, LangSmith, Pinecone) or a service (Stripe live, Shopify, Shippo live, Slack,
Telegram, SendGrid, Resend, Sentry, Grafana, New Relic, Dynatrace, PostHog, Linear, Notion,
Figma, Contentful, Typeform, Postman, ReadMe, Prefect, Mapbox); an age key; a webhook signing
secret; a Slack, Discord or Microsoft Teams webhook URL; a JWT; a password in a URL. A
placeholder passes, such as AWS's documented example key, one of x's, or a GitHub, npm,
Cloudflare or Atlassian token whose checksum fails. It names where, never the Secret itself.

To fix it, remove the Secret from the change and keep it in the environment or a secret store;
one already pushed is out, so revoke it too. Keep an Env file out of Git, or commit a template.

To mark a false alarm, put hathi:allow-secret on the line, or on an Env file's first line, whose
lines are still looked through; baloo:allow-secret and gitleaks:allow pass too. A project that
needs deeper scanning runs gitleaks beside hathi.
`
