// Package gittest makes Git repos for tests, cut off from the machine's Git: no config of the
// machine's applies, commits are unsigned, and no GIT_ variable the tests ran with, such as the
// GIT_DIR or GIT_INDEX_FILE of a Git hook that runs them, reaches Git.
package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Repo makes a new repo in a temporary folder, on the branch main, and returns its path. Until the
// test ends, Git sees only the environment Repo sets.
func Repo(t *testing.T) string {
	t.Helper()
	Isolate(t)
	dir := t.TempDir()
	Git(t, dir, "init", "-q", "-b", "main")
	return dir
}

// Isolate cuts Git off from the machine until the test ends.
func Isolate(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); strings.HasPrefix(k, "GIT_") {
			t.Setenv(k, "")
			if err := os.Unsetenv(k); err != nil {
				t.Fatal(err)
			}
		}
	}
	for k, v := range map[string]string{
		"GIT_CONFIG_GLOBAL": os.DevNull, "GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@example.com",
		"GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@example.com",
	} {
		t.Setenv(k, v)
	}
}

// Git returns what Git, run in `dir` with `args`, prints, stdout and stderr; it fails the test
// when Git fails.
func Git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// Stage writes each file of `files`, by its path from the repo `dir`, and stages it.
func Stage(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for path, text := range files {
		Write(t, dir, path, text)
		Git(t, dir, "add", path)
	}
}

// Write writes the file at `path` from `dir`, making its folders.
func Write(t *testing.T, dir, path, text string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Commit makes a commit in `dir` with `message`, empty if nothing is staged, and returns its id.
func Commit(t *testing.T, dir, message string) string {
	t.Helper()
	Git(t, dir, "commit", "-q", "--allow-empty", "-m", message)
	return strings.TrimSpace(Git(t, dir, "rev-parse", "HEAD"))
}
