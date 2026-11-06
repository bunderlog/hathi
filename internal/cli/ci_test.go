package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/bunderlog/hathi/internal/gittest"
)

// ci runs `hathi check ci --format json` on the GitHub Actions event `name` with `payload`.
func ci(t *testing.T, name string, payload map[string]any) result {
	t.Helper()
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_EVENT_NAME", name)
	text, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "event.json")
	if err := os.WriteFile(path, text, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_EVENT_PATH", path)
	return hathi("", "check", "ci", "--format", "json")
}

// checked are the commits `got` gives Verdicts for, each once, in order.
func checked(t *testing.T, got result) []string {
	t.Helper()
	var commits []string
	for line := range strings.Lines(got.stdout) {
		var v struct {
			Check, Commit string
		}
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if v.Commit != "" && !slices.Contains(commits, v.Commit) && v.Check != "linear-history" {
			commits = append(commits, v.Commit)
		}
	}
	return commits
}

func short(ids ...string) []string {
	var s []string
	for _, id := range ids {
		s = append(s, id[:12])
	}
	return s
}

// clone is a repo as a CI checkout of it has it: main with a merge commit, origin/main and
// origin's default branch main, and a branch feature from before the merge.
func clone(t *testing.T) (dir, base, merge, tip, feature string) {
	t.Helper()
	dir = repo(t)
	base = gittest.Commit(t, dir, "feat: base")
	gittest.Git(t, dir, "checkout", "-q", "-b", "side")
	gittest.Commit(t, dir, "feat: side")
	gittest.Git(t, dir, "checkout", "-q", "main")
	gittest.Git(t, dir, "merge", "-q", "--no-ff", "--no-edit", "side")
	merge = strings.TrimSpace(gittest.Git(t, dir, "rev-parse", "HEAD"))
	tip = gittest.Commit(t, dir, "feat: tip")
	gittest.Git(t, dir, "update-ref", "refs/remotes/origin/main", tip)
	gittest.Git(t, dir, "checkout", "-q", "-b", "feature", base)
	feature = gittest.Commit(t, dir, "feat: feature")
	return dir, base, merge, tip, feature
}

func push(ref, before, after string) map[string]any {
	return map[string]any{
		"ref": ref, "before": before, "after": after,
		"repository": map[string]any{"default_branch": "main"},
	}
}

var zero = strings.Repeat("0", 40)

// Criterion 3: a pull request checks from the merge base of its base and head.
func TestCIChecksAPullRequestFromTheMergeBase(t *testing.T) {
	dir, _, _, tip, feature := clone(t)
	second := gittest.Commit(t, dir, "feat: second")
	got := ci(t, "pull_request", map[string]any{"pull_request": map[string]any{
		"title": "Feature", "base": map[string]any{"sha": tip}, "head": map[string]any{"sha": second},
	}})
	if want := short(feature, second); got.code != 0 || !slices.Equal(checked(t, got), want) {
		t.Errorf("got %d, %v and %q, want 0 and %v", got.code, checked(t, got), got.stderr, want)
	}
}

// Criterion 4: with pr-title on, a title that is not a Conventional Commit fails; off, it is never
// checked.
func TestCIChecksThePullRequestTitleWithPRTitleOn(t *testing.T) {
	dir, _, _, tip, _ := clone(t)
	event := map[string]any{"pull_request": map[string]any{
		"title": "Add a feature", "base": map[string]any{"sha": tip}, "head": map[string]any{"sha": "feature"},
	}}
	if got := ci(t, "pull_request", event); got.code != 0 || strings.Contains(got.stdout, "Add a feature") {
		t.Errorf("pr-title off: got %d and %q", got.code, got.stdout)
	}
	gittest.Stage(t, dir, map[string]string{".hathi.yml": "conventional-commits: {pr-title: true}\n"})
	head := gittest.Commit(t, dir, "chore: check the PR title")
	event["pull_request"].(map[string]any)["head"] = map[string]any{"sha": head}
	got := ci(t, "pull_request", event)
	want := `{"check":"conventional-commits","verdict":"fail","reason":"\"Add a feature\" is not ` +
		"`type(scope): description`\"}\n"
	if got.code != 1 || !strings.HasSuffix(got.stdout, want) {
		t.Errorf("pr-title on: got %d and %q, want it to end in %q", got.code, got.stdout, want)
	}
	t.Setenv("GITHUB_ACTIONS", "true")
	if got := hathi("", "check", "ci", "--format", "text"); !strings.Contains(got.stderr,
		"PR title conventional-commits: \"Add a feature\"") {
		t.Errorf("got %q", got.stderr)
	}
}

// Criterion 3: a new branch checks its commits since the default branch, and a branch rebased onto
// it isn't blamed for the commits and merges it brings.
func TestCIChecksANewBranchSinceTheDefaultOne(t *testing.T) {
	dir, _, _, tip, _ := clone(t)
	gittest.Git(t, dir, "checkout", "-q", "-B", "rebased", tip)
	own := gittest.Commit(t, dir, "feat: own")
	got := ci(t, "push", push("refs/heads/rebased", zero, own))
	if want := short(own); got.code != 0 || !slices.Equal(checked(t, got), want) {
		t.Errorf("got %d, %v and %q, want 0 and %v", got.code, checked(t, got), got.stdout, want)
	}
}

// Criterion 3: after a force push, since the merge base of before and after, or since the default
// branch where the checkout lacks before.
func TestCIChecksAForcePush(t *testing.T) {
	dir, base, _, _, feature := clone(t)
	gittest.Git(t, dir, "checkout", "-q", "-B", "feature", base)
	again := gittest.Commit(t, dir, "feat: again")
	if got := ci(t, "push", push("refs/heads/feature", feature, again)); !slices.Equal(checked(t, got), short(again)) {
		t.Errorf("got %v, want %v", checked(t, got), short(again))
	}
	// Rebased onto main, it brings main's commits and merge past the merge base, but isn't
	// blamed for them.
	gittest.Git(t, dir, "checkout", "-q", "-B", "feature", "origin/main")
	rebased := gittest.Commit(t, dir, "feat: rebased")
	if got := ci(t, "push", push("refs/heads/feature", again, rebased)); got.code != 0 ||
		!slices.Equal(checked(t, got), short(rebased)) {
		t.Errorf("rebased: got %d, %v and %q", got.code, checked(t, got), got.stdout)
	}
	missing := strings.Repeat("4", 40)
	if got := ci(t, "push", push("refs/heads/feature", missing, again)); got.code != 0 ||
		!slices.Equal(checked(t, got), short(again)) {
		t.Errorf("before missing: got %d, %v and %q", got.code, checked(t, got), got.stderr)
	}
}

// Criterion 3: the default branch checks what lands on it, all of it on a first push, and exits 2
// where the checkout lacks before.
func TestCIChecksTheDefaultBranch(t *testing.T) {
	dir, base, merge, tip, _ := clone(t)
	gittest.Git(t, dir, "checkout", "-q", "main")
	got := ci(t, "push", push("refs/heads/main", base, tip))
	if got.code != 1 || !strings.Contains(got.stdout, `"check":"linear-history","verdict":"fail"`) ||
		!strings.Contains(got.stdout, merge[:12]) {
		t.Errorf("got %d and %q, want the merge to fail", got.code, got.stdout)
	}
	if got := ci(t, "push", push("refs/heads/main", zero, tip)); len(checked(t, got)) != 4 {
		t.Errorf("a first push: got %v, want all 4 commits", checked(t, got))
	}
	got = ci(t, "push", push("refs/heads/main", strings.Repeat("4", 40), tip))
	if got.code != 2 || !strings.Contains(got.stderr, "no base") {
		t.Errorf("before missing: got %d and %q, want 2", got.code, got.stderr)
	}
}

// Criterion 3: a tag or a deletion checks nothing; any other event is a usage error.
func TestCIChecksNothingForATagOrADeletion(t *testing.T) {
	_, _, _, tip, _ := clone(t)
	for _, event := range []map[string]any{
		push("refs/tags/v1.0.0", zero, tip), push("refs/heads/feature", tip, zero),
	} {
		t.Run(event["ref"].(string), func(t *testing.T) {
			if got := ci(t, "push", event); got.code != 0 || got.stdout != "" || got.stderr != "" {
				t.Errorf("got %v", got)
			}
		})
	}
	if got := ci(t, "workflow_dispatch", map[string]any{}); got.code != 2 {
		t.Errorf("another event: got %v, want 2", got)
	}
}

// Criterion 3: a shallow checkout, or a commit the checkout lacks, exits 2.
func TestCIExits2WithoutTheHistory(t *testing.T) {
	src, _, _, tip, feature := clone(t)
	shallow := filepath.Join(t.TempDir(), "shallow")
	gittest.Git(t, src, "clone", "-q", "--depth", "1", "file://"+src, shallow)
	t.Chdir(shallow)
	got := ci(t, "push", push("refs/heads/main", feature, tip))
	if got.code != 2 || !strings.Contains(got.stderr, "fetch-depth: 0") {
		t.Errorf("shallow: got %d and %q", got.code, got.stderr)
	}
	t.Chdir(src)
	got = ci(t, "pull_request", map[string]any{"pull_request": map[string]any{
		"base": map[string]any{"sha": tip}, "head": map[string]any{"sha": strings.Repeat("5", 40)},
	}})
	if got.code != 2 {
		t.Errorf("head missing: got %d and %q, want 2", got.code, got.stderr)
	}
	for _, r := range []string{tip + ".." + strings.Repeat("5", 40), strings.Repeat("5", 40) + ".." + tip} {
		if got := hathi("", "check", "--range", r); got.code != 2 {
			t.Errorf("--range %s: got %v, want 2", r, got)
		}
	}
	// Outside a Git repo, even a tag checks nothing but exits 2.
	t.Chdir(t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", os.TempDir())
	if got := ci(t, "push", push("refs/tags/v1.0.0", zero, tip)); got.code != 2 {
		t.Errorf("outside a repo: got %v, want 2", got)
	}
}
