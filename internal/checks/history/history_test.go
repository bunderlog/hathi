package history

import (
	"slices"
	"strings"
	"testing"

	"github.com/bunderlog/hathi/internal/checks"
	"github.com/bunderlog/hathi/internal/gittest"
)

var zero = strings.Repeat("0", 40)

// push is the line Git gives pre-push for pushing `local` to `branch`, where the remote had
// `remote`.
func push(branch, local, remote string) string {
	return "refs/heads/" + branch + " " + local + " refs/heads/" + branch + " " + remote + "\n"
}

// withMerge makes a branch `branch` from `start` with a merge commit of its own, and returns the
// merge.
func withMerge(t *testing.T, dir, branch, start string) string {
	t.Helper()
	gittest.Git(t, dir, "checkout", "-q", "-b", branch, start)
	gittest.Git(t, dir, "checkout", "-q", "-b", branch+"-topic")
	gittest.Commit(t, dir, "feat: topic")
	gittest.Git(t, dir, "checkout", "-q", branch)
	gittest.Git(t, dir, "merge", "-q", "--no-ff", "-m", "Merge topic", branch+"-topic")
	return strings.TrimSpace(gittest.Git(t, dir, "rev-parse", "HEAD"))
}

// history is a repo whose main has a merge commit, with the commits before it, of it and after it.
func history(t *testing.T) (dir, base, merge, after string) {
	t.Helper()
	dir = gittest.Repo(t)
	base = gittest.Commit(t, dir, "feat: base")
	gittest.Git(t, dir, "checkout", "-q", "-b", "side")
	gittest.Commit(t, dir, "feat: side")
	gittest.Git(t, dir, "checkout", "-q", "main")
	gittest.Commit(t, dir, "feat: main")
	gittest.Git(t, dir, "merge", "-q", "--no-ff", "-m", "Merge side", "side")
	merge = strings.TrimSpace(gittest.Git(t, dir, "rev-parse", "HEAD"))
	after = gittest.Commit(t, dir, "feat: after")
	return dir, base, merge, after
}

// cloned is the history repo as a clone has it: origin/main at main's tip, and origin's default
// branch main.
func cloned(t *testing.T) (dir, base, merge, after string) {
	t.Helper()
	dir, base, merge, after = history(t)
	gittest.Git(t, dir, "update-ref", "refs/remotes/origin/main", after)
	gittest.Git(t, dir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	return dir, base, merge, after
}

func find(dir, pushed string) ([]checks.Finding, error) {
	return Check.Find(checks.Input{Repo: checks.Repo{Dir: dir}, Pushed: pushed})
}

// expect checks that the push `pushed` fails for the merge commits `merges` of `branch`, none for
// a pass.
func expect(t *testing.T, dir, pushed, branch string, merges ...string) {
	t.Helper()
	got, err := find(dir, pushed)
	if err != nil {
		t.Fatal(err)
	}
	var want []checks.Finding
	for _, m := range merges {
		want = append(want, checks.Finding{Commit: m[:12], Reason: "refs/heads/" + branch})
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// baloo: test_history.py test_finds_a_merge_being_pushed
func TestCheck_FindsAMergeBeingPushed(t *testing.T) {
	dir, base, merge, _ := history(t)
	expect(t, dir, push("main", merge, base), "main", merge)
}

// baloo: test_history.py test_passes_merges_the_remote_has
func TestCheck_PassesMergesTheRemoteHas(t *testing.T) {
	dir, _, merge, after := history(t)
	expect(t, dir, push("main", after, merge), "main")
}

// A branch new to the remote, or whose remote commit isn't here, is checked against every
// remote-tracking branch.
// baloo: test_history.py test_checks_a_new_branch_against_the_remote_tracking_branches
func TestCheck_ChecksANewBranchAgainstTheRemoteTrackingBranches(t *testing.T) {
	dir, _, merge, after := history(t)
	for _, remote := range []string{zero, strings.Repeat("1", 40)} {
		t.Run(remote[:4], func(t *testing.T) { expect(t, dir, push("main", after, remote), "main", merge) })
	}
	gittest.Git(t, dir, "update-ref", "refs/remotes/origin/main", merge)
	expect(t, dir, push("main", after, zero), "main")
}

// A branch rebased onto main sends main's merges past the remote commit, but the remote's default
// branch already has them.
// baloo: test_history.py test_passes_a_rebase_onto_mains_merges
func TestCheck_PassesARebaseOntoMainsMerges(t *testing.T) {
	dir, base, _, after := cloned(t)
	gittest.Git(t, dir, "checkout", "-q", "-b", "feature", base)
	old := gittest.Commit(t, dir, "feat: feature")
	gittest.Git(t, dir, "checkout", "-q", "-B", "feature", after)
	rebased := gittest.Commit(t, dir, "feat: feature")
	expect(t, dir, push("feature", rebased, old), "feature")
}

// baloo: test_history.py test_passes_a_branch_fast_forwarded_onto_main
func TestCheck_PassesABranchFastForwardedOntoMain(t *testing.T) {
	dir, base, _, after := cloned(t)
	gittest.Git(t, dir, "checkout", "-q", "-b", "feature", after)
	ahead := gittest.Commit(t, dir, "feat: feature")
	expect(t, dir, push("feature", ahead, base), "feature")
}

// baloo: test_history.py test_finds_a_merge_a_rebased_branch_adds
func TestCheck_FindsAMergeARebasedBranchAdds(t *testing.T) {
	dir, base, _, after := cloned(t)
	gittest.Git(t, dir, "checkout", "-q", "-b", "old", base)
	old := gittest.Commit(t, dir, "feat: feature")
	own := withMerge(t, dir, "feature", after)
	expect(t, dir, push("feature", own, old), "feature", own)
}

// A merge on a remote branch other than the default one isn't on the branch the push rewrites.
// baloo: test_history.py test_finds_a_merge_from_another_remote_branch
func TestCheck_FindsAMergeFromAnotherRemoteBranch(t *testing.T) {
	dir, base, _, after := cloned(t)
	other := withMerge(t, dir, "other", base)
	gittest.Git(t, dir, "update-ref", "refs/remotes/origin/other", other)
	expect(t, dir, push("main", other, after), "main", other)
}

// The tracking ref of the branch pushed to doesn't count, past the remote commit.
// baloo: test_history.py test_finds_a_merge_rewriting_the_default_branch
func TestCheck_FindsAMergeRewritingTheDefaultBranch(t *testing.T) {
	dir, base, merge, after := cloned(t)
	gittest.Git(t, dir, "checkout", "-q", "-b", "old", base)
	old := gittest.Commit(t, dir, "feat: old")
	expect(t, dir, push("main", after, old), "main", merge)
}

// baloo: test_history.py test_passes_a_linear_history
func TestCheck_PassesALinearHistory(t *testing.T) {
	dir, base, _, _ := history(t)
	expect(t, dir, push("main", base, zero), "main")
}

// baloo: test_history.py test_ignores_deletions_and_blank_lines
func TestCheck_IgnoresDeletionsAndBlankLines(t *testing.T) {
	dir, base, _, _ := history(t)
	expect(t, dir, "\nrefs/heads/x "+zero+" refs/heads/x "+base+"\n", "x")
}

// baloo: test_history.py test_fails_on_a_commit_not_here
func TestCheck_FailsOnACommitNotHere(t *testing.T) {
	dir, base, _, _ := history(t)
	if _, err := find(dir, push("main", strings.Repeat("2", 40), base)); err == nil {
		t.Error("got no error for a commit not here")
	}
}
