package conflict

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/bunderlog/hathi/internal/checks"
	"github.com/bunderlog/hathi/internal/gittest"
)

// Built at run time, so this file holds no line that opens a conflict.
var open, middle, closing = strings.Repeat("<", 7), strings.Repeat("=", 7), strings.Repeat(">", 7)

func find(dir string) ([]checks.Finding, error) {
	return Check.Find(checks.Input{Changes: checks.Changes{Repo: checks.Repo{Dir: dir}}})
}

func expect(t *testing.T, dir string, want ...checks.Finding) {
	t.Helper()
	got, err := find(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func marker(line int) checks.Finding {
	return checks.Finding{File: "a.go", Line: line, Reason: "a conflict marker"}
}

// baloo: test_conflict.py test_finds_each_marker_the_staged_changes_add
func TestFindsEachMarkerTheStagedChangesAdd(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Stage(t, dir, map[string]string{
		"a.go": "a\n" + open + " HEAD\nx \n" + middle + "\ny\n" + closing + " topic\n",
	})
	expect(t, dir, marker(2), marker(4), marker(6))
}

// baloo: test_conflict.py test_passes_what_is_not_a_marker
func TestPassesWhatIsNotAMarker(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Stage(t, dir, map[string]string{
		"a.md":  "Title\n" + middle + "\n\ntrailing space \n",
		"b.txt": open + "< eight, not seven\n",
	})
	expect(t, dir)
}

// A file that holds markers on purpose passes with a longer conflict-marker-size in
// .gitattributes, as Git's own check reads it.
// baloo: test_conflict.py test_passes_a_file_with_a_longer_marker_size
func TestPassesAFileWithALongerMarkerSize(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Write(t, dir, ".gitattributes", "testdata/* conflict-marker-size=32\n")
	gittest.Stage(t, dir, map[string]string{"testdata/merge.txt": open + " ours\n"})
	expect(t, dir)
}

// baloo: test_conflict.py test_passes_nothing_staged
func TestPassesNothingStaged(t *testing.T) {
	expect(t, gittest.Repo(t))
}

// baloo: test_conflict.py test_fails_outside_a_repo
func TestFailsOutsideARepo(t *testing.T) {
	gittest.Isolate(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", os.TempDir())
	if _, err := find(t.TempDir()); err == nil {
		t.Error("got no error outside a repo")
	}
}

// A commit's changes, as a range checks them, are read with the commit's own .gitattributes.
func TestReadsACommitWithItsOwnAttributes(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Stage(t, dir, map[string]string{"a.go": open + " ours\n"})
	commit := gittest.Commit(t, dir, "feat: a")
	gittest.Write(t, dir, ".gitattributes", "a.go conflict-marker-size=32\n")
	got, err := Check.Find(checks.Input{
		Changes: checks.Changes{Repo: checks.Repo{Dir: dir}, Commit: commit},
	})
	if err != nil || !slices.Equal(got, []checks.Finding{marker(1)}) {
		t.Errorf("got %v and %v, want a.go:1", got, err)
	}
}

// A Git failure on a commit's changes names the subcommand that failed, past --attr-source.
func TestNamesTheGitSubcommandThatFailed(t *testing.T) {
	dir := gittest.Repo(t)
	_, err := find(dir + "/missing")
	if err == nil || !strings.HasPrefix(err.Error(), "git diff: ") {
		t.Errorf("got %v, want it to name git diff", err)
	}
	_, err = Check.Find(checks.Input{Changes: checks.Changes{
		Repo: checks.Repo{Dir: dir}, Commit: strings.Repeat("6", 40), Parent: strings.Repeat("7", 40),
	}})
	if err == nil || !strings.HasPrefix(err.Error(), "git diff: ") {
		t.Errorf("got %v, want it to name git diff", err)
	}
}
