// Package history is the Check linear-history.
package history

import (
	"strings"

	"github.com/bunderlog/hathi/internal/checks"
)

// Check is linear-history: each merge commit the push sends, given as Git gives pre-push its
// stdin, a `<local ref> <local sha> <remote ref> <remote sha>` line per ref. A ref new to the
// remote, or whose remote commit isn't here, is checked against every remote-tracking branch.
// Any other is checked against its remote commit and each remote's default branch, which has the
// merges a branch rebased onto it brings along; that remote's tracking ref of the branch pushed
// to doesn't count, as the push replaces it. A deletion sends no commits. Over a Range, it is each
// merge commit of the Range, by the Range's head.
var Check = checks.Check{
	Name:    "linear-history",
	Hook:    "pre-push",
	Explain: explanation,
	Find: func(in checks.Input) ([]checks.Finding, error) {
		if in.Range != nil {
			return inRange(in.Repo, *in.Range)
		}
		defaults, err := in.Repo.Git("for-each-ref", "--format=%(refname) %(symref)", "refs/remotes/*/HEAD")
		if err != nil {
			return nil, err
		}
		var found []checks.Finding
		for line := range strings.SplitSeq(in.Pushed, "\n") {
			fields := strings.Fields(line)
			if len(fields) < 4 || checks.IsZero(fields[1]) {
				continue
			}
			ref, local, to, remote := fields[0], fields[1], fields[2], fields[3]
			sends := []string{local, "--not", "--remotes"}
			if !checks.IsZero(remote) {
				if _, err := in.Repo.Git("cat-file", "-e", remote+"^{commit}"); err == nil {
					sends = []string{local, "--not", remote}
					for d := range strings.SplitSeq(strings.TrimSpace(defaults), "\n") {
						head, branch, _ := strings.Cut(d, " ")
						pushedTo := strings.TrimSuffix(head, "HEAD") + strings.TrimPrefix(to, "refs/heads/")
						if branch != "" && branch != pushedTo {
							sends = append(sends, branch)
						}
					}
				}
			}
			merges, err := in.Repo.Git(append([]string{"rev-list", "--merges"}, sends...)...)
			if err != nil {
				return nil, err
			}
			for commit := range strings.FieldsSeq(merges) {
				found = append(found, checks.Finding{Commit: commit[:12], Reason: ref})
			}
		}
		return found, nil
	},
}

// inRange finds each merge commit of the Range `r`, oldest first.
func inRange(repo checks.Repo, r checks.Range) ([]checks.Finding, error) {
	head, err := repo.Git("rev-parse", r.Head+"^{commit}")
	if err != nil {
		return nil, err
	}
	args := []string{"rev-list", "--merges", "--reverse", r.Head}
	if r.From != "" {
		args = append(args, "^"+r.From)
	}
	if len(r.Not) > 0 {
		args = append(append(args, "--not"), r.Not...)
	}
	merges, err := repo.Git(args...)
	if err != nil {
		return nil, err
	}
	var found []checks.Finding
	for commit := range strings.FieldsSeq(merges) {
		found = append(found, checks.Finding{Commit: commit[:12], Reason: strings.TrimSpace(head)[:12]})
	}
	return found, nil
}

// explanation is what `hathi explain` prints for the Check: what it looks for, how to fix a
// failure, and how to mark a false alarm.
const explanation = `linear-history (pre-push) looks for a merge commit the push sends, and over
a range for a merge commit in it; the merges the remote's default branch already has pass.

To fix it, rebase instead of merging (git rebase <branch>), then push the rebased branch with
git push --force-with-lease. git config pull.rebase true makes git pull rebase.

linear-history: false in .hathi.yml turns it off.
`
