package size

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/bunderlog/hathi/internal/checks"
	"github.com/bunderlog/hathi/internal/gittest"
)

func find(dir string, maxSize int) ([]checks.Finding, error) {
	return Check(maxSize).Find(checks.Input{Changes: checks.Changes{Repo: checks.Repo{Dir: dir}}})
}

func expect(t *testing.T, dir string, maxSize int, want ...checks.Finding) {
	t.Helper()
	got, err := find(dir, maxSize)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// baloo: test_size.py test_finds_an_added_file_over_the_limit
func TestCheck_FindsAnAddedFileOverTheLimit(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Stage(t, dir, map[string]string{
		"bin/app": strings.Repeat("x", 3*1024+1), "small.go": strings.Repeat("x", 3*1024),
	})
	expect(t, dir, 3, checks.Finding{File: "bin/app", Reason: "4 KB, over 3 KB"})
}

// Without a limit of its own, a file may be 1 MB.
// baloo: test_size.py test_limits_a_file_to_1_mb_by_default
func TestCheck_LimitsAFileTo1MBByDefault(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Stage(t, dir, map[string]string{"a": strings.Repeat("x", 1024*1024)})
	expect(t, dir, DefaultMaxSize)
	gittest.Stage(t, dir, map[string]string{"b": strings.Repeat("x", 1024*1024+1)})
	expect(t, dir, DefaultMaxSize, checks.Finding{File: "b", Reason: "1025 KB, over 1024 KB"})
}

// A file already committed passes when it changes or moves: it was let in once.
// baloo: test_size.py test_passes_a_file_already_committed
func TestCheck_PassesAFileAlreadyCommitted(t *testing.T) {
	dir := gittest.Repo(t)
	big := strings.Repeat("x", 4*1024)
	gittest.Stage(t, dir, map[string]string{"data.bin": big})
	gittest.Commit(t, dir, "chore: data")
	gittest.Stage(t, dir, map[string]string{"data.bin": big + "y"})
	gittest.Git(t, dir, "mv", "data.bin", "moved.bin")
	expect(t, dir, 1)
}

// baloo: test_size.py test_fails_outside_a_repo
func TestCheck_FailsOutsideARepo(t *testing.T) {
	gittest.Isolate(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", os.TempDir())
	if _, err := find(t.TempDir(), DefaultMaxSize); err == nil {
		t.Error("got no error outside a repo")
	}
}

// A commit's changes, as a range checks them, are against its parent, or nothing for a root
// commit.
func TestCheck_ReadsACommitAgainstItsParent(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Stage(t, dir, map[string]string{"a": strings.Repeat("x", 2048)})
	root := gittest.Commit(t, dir, "feat: a")
	gittest.Stage(t, dir, map[string]string{"b": strings.Repeat("x", 2048)})
	next := gittest.Commit(t, dir, "feat: b")
	for commit, want := range map[string]string{root: "a", next: "b"} {
		got, err := Check(1).Find(checks.Input{
			Changes: checks.Changes{Repo: checks.Repo{Dir: dir}, Commit: commit},
		})
		if err != nil || !slices.Equal(got, []checks.Finding{{File: want, Reason: "2 KB, over 1 KB"}}) {
			t.Errorf("%s: got %v and %v, want %s", want, got, err, want)
		}
	}
}
