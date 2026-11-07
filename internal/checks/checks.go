// Package checks holds what every Check shares: the interface each one is behind, what it reads
// and what it finds.
package checks

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// A Check is one deterministic rule over a commit or the repo that passes or fails (glossary).
type Check struct {
	Name string // such as conventional-commits
	Hook string // the Git hook that runs it: commit-msg, pre-commit or pre-push
	// Explain is what `hathi explain` prints: what the Check looks for, how to fix a failure, and
	// how to mark a false alarm.
	Explain string
	// Find returns what the Check fails on in `in`, none when it passes, or an error when it can't
	// check.
	Find func(in Input) ([]Finding, error)
}

// Input is what a Check reads.
type Input struct {
	Message string  // the commit message, for a commit-msg Check, as Git wrote it to its file
	Changes Changes // the changes, for a pre-commit Check
	Pushed  string  // the pushed refs, for a pre-push Check, as Git gives them on stdin
	Repo    Repo    // the repo, for a pre-push Check
}

// A Finding is one thing a Check fails on: where, when the Check names a place, and why.
type Finding struct {
	File   string // "" for none
	Line   int    // 1-based; 0 for none
	Commit string // 12 characters; "" for none
	Reason string
}

// Repo runs Git in a repo.
type Repo struct {
	Dir string // where Git runs; "" for the current folder
}

// Git returns what Git, run with `args`, prints on stdout, or an error with why it failed.
func (r Repo) Git(args ...string) (string, error) {
	return r.GitOK(nil, "", args...)
}

// GitOK is Git with `stdin`, where an exit code of `ok` counts as success too.
func (r Repo) GitOK(ok []int, stdin string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-c", "core.quotePath=false"}, args...)...)
	cmd.Dir = r.Dir
	cmd.Stdin = strings.NewReader(stdin)
	// Git's own words, which some output is told by.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		// The subcommand, past options such as --attr-source.
		sub := args[max(0, slices.IndexFunc(args, func(a string) bool { return !strings.HasPrefix(a, "-") }))]
		exit, isExit := err.(*exec.ExitError)
		if !isExit {
			return "", fmt.Errorf("git %s: %w", sub, err)
		}
		if !slices.Contains(ok, exit.ExitCode()) {
			return "", fmt.Errorf("git %s: exit status %d: %s", sub, exit.ExitCode(),
				strings.TrimSpace(stderr.String()))
		}
	}
	return stdout.String(), nil
}

// Changes are the changes a pre-commit Check looks at: the staged ones, or a commit's against
// its parent, as if staged there.
type Changes struct {
	Repo
	Commit string // "" for the staged changes
	// Parent is the commit's first parent, or the empty tree for a root commit; "" to look it up.
	Parent string
}

// Diff returns what `git diff` with `args` prints for the changes: for a commit's, against its
// first parent, or the empty tree for a root commit, with the commit's .gitattributes. An exit
// code of `ok` counts as success too.
func (c Changes) Diff(ok []int, args ...string) (string, error) {
	if c.Commit == "" {
		return c.GitOK(ok, "", append([]string{"diff", "--cached"}, args...)...)
	}
	parent := c.Parent
	if parent == "" {
		var err error
		if parent, err = c.parent(); err != nil {
			return "", err
		}
	}
	return c.GitOK(ok, "", append(append([]string{"--attr-source=" + c.Commit, "diff"}, args...),
		parent, c.Commit)...)
}

// File returns the file at `path` as the changes leave it.
func (c Changes) File(path string) (string, error) {
	return c.Git("cat-file", "blob", c.Commit+":"+path)
}

func (c Changes) parent() (string, error) {
	out, err := c.Git("rev-list", "--parents", "-n", "1", c.Commit)
	if err != nil {
		return "", err
	}
	if fields := strings.Fields(out); len(fields) > 1 {
		return fields[1], nil
	}
	return c.EmptyTree()
}

// EmptyTree is the id of the empty tree, which a root commit's changes are against.
func (r Repo) EmptyTree() (string, error) {
	out, err := r.GitOK(nil, "", "hash-object", "-t", "tree", "--stdin")
	return strings.TrimSpace(out), err
}

// Where `git commit -v` puts the diff below the message.
var scissors = regexp.MustCompile(`(?m)^# -+ >8 -+$`)

// AboveScissors returns the commit message above a `git commit -v` scissors line, all of it
// without one.
func AboveScissors(message string) string {
	if loc := scissors.FindStringIndex(message); loc != nil {
		return message[:loc[0]]
	}
	return message
}

// Space is what Python's str.isspace counts as whitespace, to put in a character class: patterns
// ported from `baloo` mean Python's \s, and RE2's is ASCII only and leaves out \v.
const Space = `\t-\r\x{1c}-\x{1f}\x{85}\p{Z}`

// IsSpace reports whether Python's str.isspace counts r as whitespace.
func IsSpace(r rune) bool {
	return unicode.IsSpace(r) || unicode.Is(unicode.Z, r) || (r >= 0x1c && r <= 0x1f)
}

// TrimSpace is Python's str.strip.
func TrimSpace(s string) string {
	return strings.TrimFunc(s, IsSpace)
}

// IsZero reports whether `id` is Git's id of no commit, all zeros, as a push gives a new ref's
// remote side or a deletion's local one.
func IsZero(id string) bool {
	return strings.Trim(id, "0") == ""
}

// Lines splits `text` at "\n" without the empty string after a last one, as Python's
// str.removesuffix("\n").split("\n") does.
func Lines(text string) []string {
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}
