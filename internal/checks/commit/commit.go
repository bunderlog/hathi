// Package commit is the Check conventional-commits.
package commit

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bunderlog/hathi/internal/checks"
)

// A letter or a digit, as Python's `[^\W_]` in `baloo`.
const alnum = `[\p{L}\p{N}]`

var (
	subject = regexp.MustCompile(
		`^` + alnum + `(?:` + alnum + `|-)*(\([^()\n]*[^()` + checks.Space + `][^()\n]*\))?!?: [^` +
			checks.Space + `]`)
	header = regexp.MustCompile(`^([^(!:]*)(?:\(([^()]*)\))?!?: (.*)$`)
	// A footer's word token (`-` for spaces; BREAKING CHANGE the one exception) and its separator.
	footer   = regexp.MustCompile(`^((?:` + alnum + `|-)+|BREAKING CHANGE)(: | #)`)
	breaking = regexp.MustCompile(`(?i)^breaking[ -]change$`)
	// A message Git, or a pull request's merge, writes itself.
	byGit = regexp.MustCompile(
		`^Merge (branch|branches|remote-tracking branch|tag|commit|pull request) |` +
			`^(Revert|Reapply) "|^(fixup|squash|amend)! `)
	newline   = regexp.MustCompile(`\r?\n`)
	paragraph = regexp.MustCompile(`\n[` + checks.Space + `]*\n`)
	// What separates the scopes in one, as commitlint's scope-enum splits it.
	scopeDelimiter = regexp.MustCompile(`/|\\|, ?`)
)

// Types are config-conventional's types.
var Types = []string{
	"build", "chore", "ci", "docs", "feat", "fix", "perf", "refactor", "revert", "style", "test",
}

// Rules are conventional-commits' settings.
type Rules struct {
	Types     []string // the types a subject may have; nil for any
	Scopes    []string // the scopes a subject may have; nil for any
	MaxLength int      // the longest line, in characters; 0 for no limit
}

// Default is config-conventional's Rules.
var Default = Rules{Types: Types, MaxLength: 100}

// Check is conventional-commits with `rules`.
func Check(rules Rules) checks.Check {
	return checks.Check{
		Name: "conventional-commits",
		Hook: "commit-msg",
		Find: func(in checks.Input) ([]checks.Finding, error) {
			if reason := why(in.Message, rules); reason != "" {
				return []checks.Finding{{Reason: reason}}, nil
			}
			return nil, nil
		},
	}
}

// why says why `message` isn't a Conventional Commit as commitlint's config-conventional reads it
// (its error rules), or "" when it is. That is a `type(scope)!: description` subject with one of
// its types and scopes, a description not starting in uppercase (quoted text aside) nor ending in
// `.`, a trimmed header, a body after a blank line, lines of up to 100 characters, and a breaking
// change footer as `BREAKING CHANGE: description`, in uppercase (its synonym `BREAKING-CHANGE`
// too). Footers are the trailing paragraphs that open with one. Lines starting with `#` don't
// count, and a message Git writes itself, such as a merge's, passes.
func why(message string, rules Rules) string {
	var lines []string
	for _, line := range newline.Split(checks.AboveScissors(message), -1) {
		if !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	start := slices.IndexFunc(lines, func(line string) bool { return checks.TrimSpace(line) != "" })
	if start < 0 || byGit.MatchString(lines[start]) {
		return ""
	}
	lines = lines[start:]
	s := lines[0]
	if !subject.MatchString(s) {
		return fmt.Sprintf("\"%s\" is not `type(scope): description`", s)
	}
	m := header.FindStringSubmatch(s)
	kind, scope, description := m[1], m[2], m[3]
	if rules.Types != nil && !slices.Contains(rules.Types, kind) {
		return fmt.Sprintf("\"%s\" is not one of %s", kind, strings.Join(rules.Types, ", "))
	}
	if why := whyScope(scope, rules.Scopes); why != "" {
		return why
	}
	// A description that opens with quoted text, such as `README`, may be in uppercase.
	if description != "" && description[0] >= 'A' && description[0] <= 'Z' {
		return fmt.Sprintf("\"%s\" starts with an uppercase letter", description)
	}
	if strings.HasSuffix(description, ".") {
		return fmt.Sprintf("\"%s\" ends with a full stop", description)
	}
	if s != checks.TrimSpace(s) {
		return fmt.Sprintf("\"%s\" has trailing spaces", s)
	}
	if len(lines) > 1 && checks.TrimSpace(lines[1]) != "" {
		return fmt.Sprintf("leave a blank line between \"%s\" and the body", s)
	}
	for _, line := range lines {
		if n := utf8.RuneCountInString(line); rules.MaxLength > 0 && n > rules.MaxLength {
			return fmt.Sprintf("\"%s…\" is %d characters, over %d", prefix(line, 40), n, rules.MaxLength)
		}
	}
	paragraphs := paragraph.Split(checks.TrimSpace(strings.Join(lines[1:], "\n")), -1)
	var footers []string
	for _, p := range slices.Backward(paragraphs) {
		if !footer.MatchString(p) {
			break // the body
		}
		footers = append(footers, strings.Split(p, "\n")...)
	}
	for _, line := range footers {
		m := footer.FindStringSubmatch(line)
		if m != nil && breaking.MatchString(m[1]) && (m[1] != strings.ToUpper(m[1]) || m[2] != ": ") {
			return fmt.Sprintf("\"%s\" is not `BREAKING CHANGE: description`", line)
		}
	}
	return ""
}

// whyScope says why `scope` isn't one of `scopes`, nil for any, or "" when it is. As commitlint's
// scope-enum does, a scope such as `api,ui` is each of its parts, and it passes as a whole too.
func whyScope(scope string, scopes []string) string {
	if scope == "" || scopes == nil || slices.Contains(scopes, scope) {
		return ""
	}
	for _, part := range scopeDelimiter.Split(scope, -1) {
		if !slices.Contains(scopes, part) {
			return fmt.Sprintf("\"%s\" is not one of %s", part, strings.Join(scopes, ", "))
		}
	}
	return ""
}

// prefix is the first `n` characters of `s`.
func prefix(s string, n int) string {
	for i := range s {
		if n == 0 {
			return s[:i]
		}
		n--
	}
	return s
}
