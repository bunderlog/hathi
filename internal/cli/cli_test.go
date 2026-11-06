package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bunderlog/hathi/internal/gittest"
	"github.com/bunderlog/hathi/internal/settings"
)

// A Secret built at run time, so this file holds none.
var aws = "AKIA" + "IOSFODNN7EXAMPLE"

// repo makes a repo of its own for the test and runs the test in it, outside GitHub Actions.
func repo(t *testing.T) string {
	t.Helper()
	dir := gittest.Repo(t)
	t.Chdir(dir)
	t.Setenv("GITHUB_ACTIONS", "")
	return dir
}

type result struct {
	code           int
	stdout, stderr string
}

// hathi runs it with `args` and `stdin`.
func hathi(stdin string, args ...string) result {
	var stdout, stderr strings.Builder
	code := Run("", args, strings.NewReader(stdin), &stdout, &stderr)
	return result{code, stdout.String(), stderr.String()}
}

func expect(t *testing.T, got result, code int, stderr string) {
	t.Helper()
	if got.code != code || got.stderr != stderr {
		t.Errorf("got %d and stderr %q, want %d and %q", got.code, got.stderr, code, stderr)
	}
}

// Criterion 1.
func TestCommitMsgFailsAMessageWithALinePerFailure(t *testing.T) {
	dir := repo(t)
	message := "add login\n\nCo-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>\n"
	want := "conventional-commits: \"add login\" is not `type(scope): description`; " +
		"run: hathi explain conventional-commits\n" +
		"no-ai-coauthor: 3: Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>; " +
		"run: hathi explain no-ai-coauthor\n"
	gittest.Write(t, dir, "MSG", message)
	expect(t, hathi("", "check", "commit-msg", "MSG"), 1, want)
	expect(t, hathi(message, "check", "commit-msg", "-"), 1, want)
}

// Criterion 1.
func TestCommitMsgPassesAMessageQuietly(t *testing.T) {
	dir := repo(t)
	message := "feat: add login\n\nCo-authored-by: Jane Doe <jane@example.com>\n"
	gittest.Write(t, dir, "MSG", message)
	expect(t, hathi("", "check", "commit-msg", "MSG"), 0, "")
	expect(t, hathi(message, "check", "commit-msg", "-"), 0, "")
}

// Criterion 2.
func TestPreCommitChecksTheStagedChanges(t *testing.T) {
	dir := repo(t)
	gittest.Stage(t, dir, map[string]string{"a.go": "ok\n"})
	expect(t, hathi("", "check", "pre-commit"), 0, "")
	gittest.Stage(t, dir, map[string]string{"b.go": "key := " + aws + "\n", ".env": "A=1\n"})
	expect(t, hathi("", "check", "pre-commit"), 1,
		"no-secrets-in-commits: .env: an Env file; keep it out of Git, or put hathi:allow-secret on its first line; "+
			"run: hathi explain no-secrets-in-commits\n"+
			"no-secrets-in-commits: b.go:1: an AWS access key; run: hathi explain no-secrets-in-commits\n")
}

// Criterion 2: with or without <remote> <url>, whatever they are.
func TestPrePushChecksThePushedRefs(t *testing.T) {
	dir := repo(t)
	base := gittest.Commit(t, dir, "feat: base")
	gittest.Git(t, dir, "checkout", "-q", "-b", "side")
	gittest.Commit(t, dir, "feat: side")
	gittest.Git(t, dir, "checkout", "-q", "main")
	gittest.Commit(t, dir, "feat: main")
	gittest.Git(t, dir, "merge", "-q", "--no-ff", "-m", "Merge side", "side")
	merge := strings.TrimSpace(gittest.Git(t, dir, "rev-parse", "HEAD"))
	pushed := "refs/heads/main " + merge + " refs/heads/main " + base + "\n"
	want := "linear-history: refs/heads/main; run: hathi explain linear-history\n"
	want = merge[:12] + " " + want
	for _, args := range [][]string{{}, {"origin", "git@example.com:a/b.git"}, {"x", "y"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			expect(t, hathi(pushed, append([]string{"check", "pre-push"}, args...)...), 1, want)
		})
	}
	expect(t, hathi("refs/heads/main "+base+" refs/heads/main "+base+"\n", "check", "pre-push"),
		0, "")
}

// rangeRepo is a repo whose main has a commit that adds a Secret, one that removes it, and a merge.
func rangeRepo(t *testing.T) (dir, base, adds, removes, merge string) {
	t.Helper()
	dir = repo(t)
	base = gittest.Commit(t, dir, "feat: base")
	gittest.Stage(t, dir, map[string]string{"a.go": "key := " + aws + "\n"})
	adds = gittest.Commit(t, dir, "feat: add a key")
	gittest.Stage(t, dir, map[string]string{"a.go": "key := os.Getenv(\"KEY\")\n"})
	removes = gittest.Commit(t, dir, "Remove the key")
	gittest.Git(t, dir, "checkout", "-q", "-b", "side", base)
	gittest.Commit(t, dir, "feat: side")
	gittest.Git(t, dir, "checkout", "-q", "main")
	gittest.Git(t, dir, "merge", "-q", "--no-ff", "--no-edit", "side")
	merge = strings.TrimSpace(gittest.Git(t, dir, "rev-parse", "HEAD"))
	return dir, base, adds, removes, merge
}

// Criterion 3: each commit's message and changes, and its merge commits; a Secret one commit adds
// and the next removes fails.
func TestRangeChecksEachCommit(t *testing.T) {
	_, base, adds, removes, merge := rangeRepo(t)
	expect(t, hathi("", "check", "--range", base+"..main"), 1,
		adds[:12]+" no-secrets-in-commits: a.go:1: an AWS access key; "+
			"run: hathi explain no-secrets-in-commits\n"+
			removes[:12]+" conventional-commits: \"Remove the key\" is not `type(scope): description`; "+
			"run: hathi explain conventional-commits\n"+
			merge[:12]+" linear-history: "+merge[:12]+"; run: hathi explain linear-history\n")
}

// Criterion 5: one JSON object per Verdict, passing ones too.
func TestRangePrintsEachVerdictInJSON(t *testing.T) {
	_, base, adds, _, _ := rangeRepo(t)
	got := hathi("", "check", "--range", base+".."+adds, "--format", "json")
	var verdicts []map[string]any
	for line := range strings.Lines(got.stdout) {
		var v map[string]any
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		verdicts = append(verdicts, v)
	}
	want := []string{
		"conventional-commits pass", "no-ai-coauthor pass", "no-secrets-in-commits fail",
		"no-conflict-markers pass", "no-large-files pass", "linear-history pass",
	}
	if got.code != 1 || len(verdicts) != len(want) {
		t.Fatalf("got %d and %v, want 1 and %v", got.code, verdicts, want)
	}
	for i, v := range verdicts {
		if s := v["check"].(string) + " " + v["verdict"].(string); s != want[i] {
			t.Errorf("verdict %d: got %s, want %s", i, s, want[i])
		}
	}
	if v := verdicts[2]; v["commit"] != adds[:12] || v["file"] != "a.go" || v["line"] != 1.0 ||
		v["reason"] != "an AWS access key" {
		t.Errorf("got %v", v)
	}
}

// Criterion 7: over a range, a commit whose .hathi.yml turns a Check off is checked with it off,
// and the commits before it with it on.
func TestRangeChecksEachCommitWithItsOwnSettings(t *testing.T) {
	dir := repo(t)
	base := gittest.Commit(t, dir, "feat: base")
	before := gittest.Commit(t, dir, "Before")
	gittest.Stage(t, dir, map[string]string{".hathi.yml": "conventional-commits: false\n"})
	gittest.Commit(t, dir, "Off")
	gittest.Commit(t, dir, "After")
	expect(t, hathi("", "check", "--range", base+"..main"), 1, before[:12]+
		" conventional-commits: \"Before\" is not `type(scope): description`; "+
		"run: hathi explain conventional-commits\n")
}

// Criterion 6.
func TestUsageErrorsExit2(t *testing.T) {
	repo(t)
	for _, args := range [][]string{
		{}, {"check"}, {"check", "commit-msg"}, {"check", "commit-msg", "a", "b"},
		{"check", "post-merge"}, {"lint"}, {"check", "pre-commit", "--only", "no-large-files",
			"--skip", "no-conflict-markers"},
		{"check", "pre-commit", "--only", "no-such-check"},
		{"check", "commit-msg", "-", "--only", "no-large-files"},
		{"check", "ci"}, {"check", "--range", "main"}, {"check", "pre-commit", "--range", "a..b"},
		{"check", "pre-commit", "--format", "xml"}, {"explain", "no-such-check"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if got := hathi("", args...); got.code != 2 || !strings.HasPrefix(got.stderr, "hathi: ") {
				t.Errorf("got %d and %q, want 2", got.code, got.stderr)
			}
		})
	}
	if got := hathi("", "check", "ci"); !strings.Contains(got.stderr, "--range") {
		t.Errorf("check ci outside GitHub Actions: got %q, want it to name --range", got.stderr)
	}
}

// Criterion 6: a Check that can't check exits 2.
func TestCantCheckExits2(t *testing.T) {
	repo(t)
	expect(t, hathi("", "check", "commit-msg", "missing"), 2,
		"hathi: open missing: no such file or directory\n")
	got := hathi("refs/heads/main "+strings.Repeat("2", 40)+" refs/heads/main "+
		strings.Repeat("3", 40)+"\n", "check", "pre-push")
	if got.code != 2 {
		t.Errorf("a push of a commit not here: got %d and %q, want 2", got.code, got.stderr)
	}
	t.Chdir(t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", os.TempDir())
	expect(t, hathi("feat: x\n", "check", "commit-msg", "-"), 2, "hathi: not in a Git repo\n")
}

// Criterion 6: --only runs a Check the settings turn off, --skip leaves one out.
func TestOnlyAndSkip(t *testing.T) {
	dir := repo(t)
	gittest.Write(t, dir, ".hathi.yml", "conventional-commits: false\n")
	bad := "Add login\n\nGenerated with Claude Code\n"
	expect(t, hathi(bad, "check", "commit-msg", "-", "--skip", "no-ai-coauthor"), 0, "")
	got := hathi(bad, "check", "commit-msg", "-", "--only", "conventional-commits")
	if got.code != 1 || !strings.HasPrefix(got.stderr, "conventional-commits: ") ||
		strings.Count(got.stderr, "\n") != 1 {
		t.Errorf("got %d and %q, want conventional-commits alone", got.code, got.stderr)
	}
}

// Criterion 7.
func TestSettingsComeFromTheRootOrConfig(t *testing.T) {
	dir := repo(t)
	gittest.Write(t, dir, ".hathi.yml", "conventional-commits: {types: [wip]}\n")
	gittest.Write(t, dir, "ci.yml", "conventional-commits: false\n")
	expect(t, hathi("wip: x\n", "check", "commit-msg", "-"), 0, "")
	expect(t, hathi("Anything\n", "check", "commit-msg", "-", "--config", "ci.yml"), 0, "")
	expect(t, hathi("x\n", "check", "commit-msg", "-", "--config", "none.yml"), 2,
		"hathi: open none.yml: no such file or directory\n")
	gittest.Write(t, dir, ".hathi.yml", "linear-history: true\nno-large-file: true\n")
	expect(t, hathi("feat: x\n", "check", "commit-msg", "-"), 2,
		"hathi: .hathi.yml:2: \"no-large-file\" is not a setting\n")
}

// Criterion 7: hathi config --default prints exactly the defaults.
func TestConfigDefaultPrintsTheDefaults(t *testing.T) {
	got := hathi("", "config", "--default")
	s, problems := settings.Parse(got.stdout)
	if got.code != 0 || problems != nil || len(s.Off) > 0 ||
		!strings.HasPrefix(got.stdout, "# yaml-language-server: $schema="+settings.Schema+"\n") {
		t.Errorf("got %d, %v and %q", got.code, problems, got.stdout)
	}
}

// Criterion 5: annotations by default in GitHub Actions.
func TestGitHubAnnotations(t *testing.T) {
	dir := repo(t)
	t.Setenv("GITHUB_ACTIONS", "true")
	gittest.Stage(t, dir, map[string]string{"a,b.go": "key := " + aws + "\n"})
	got := hathi("", "check", "pre-commit")
	if want := "::error file=a%2Cb.go,line=1,title=no-secrets-in-commits::an AWS access key\n"; got.code != 1 || got.stdout != want {
		t.Errorf("got %d and %q, want 1 and %q", got.code, got.stdout, want)
	}
	got = hathi("Add x\n", "check", "commit-msg", "-")
	if want := "::error title=conventional-commits::\"Add x\" is not `type(scope): description`\n"; got.stdout != want {
		t.Errorf("got %q, want %q", got.stdout, want)
	}
	got = hathi("", "check", "--range", "nothing..main")
	if got.code != 2 || !strings.HasPrefix(got.stdout, "::error title=hathi::") {
		t.Errorf("an error: got %d and %q, want 2 and an annotation", got.code, got.stdout)
	}
}

// Criterion 8.
func TestListAndDoctorChangeNothing(t *testing.T) {
	dir := repo(t)
	gittest.Commit(t, dir, "feat: x")
	gittest.Write(t, dir, ".hathi.yml", "linear-history: false\n")
	before := gittest.Git(t, dir, "status", "--porcelain", "--ignored") + gittest.Git(t, dir, "config", "--list", "--local")
	got := hathi("", "list")
	if got.code != 0 || !strings.Contains(got.stdout, "linear-history") ||
		!strings.Contains(got.stdout, "types: [build, chore") {
		t.Errorf("list: got %d and %q", got.code, got.stdout)
	}
	if got := hathi("", "doctor"); got.code != 1 {
		t.Errorf("doctor with no Git hooks: got %d and %q, want 1", got.code, got.stdout)
	}
	after := gittest.Git(t, dir, "status", "--porcelain", "--ignored") + gittest.Git(t, dir, "config", "--list", "--local")
	if before != after {
		t.Errorf("got %q, want %q", after, before)
	}
}

// Criterion 8: doctor exits 0 when it reports no problem, 2 when one is in the settings.
func TestDoctorFindsTheGitHooks(t *testing.T) {
	dir := repo(t)
	gittest.Write(t, dir, ".hathi.yml", "no-ai-coauthor: false\n")
	hooks := filepath.Join(dir, ".git", "hooks")
	for _, hook := range Hooks {
		script := hathi("", "hooks", "print", hook).stdout
		if hook == "pre-push" {
			script = "#!/bin/sh\nmise exec -- hathi check pre-push \"$@\"\n"
		}
		if err := os.WriteFile(filepath.Join(hooks, hook), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got := hathi("", "doctor", "--format", "json")
	if got.code != 0 || strings.Contains(got.stdout, `"problem":true`) {
		t.Errorf("got %d and %s, want 0", got.code, got.stdout)
	}
	if err := os.Chmod(filepath.Join(hooks, "pre-commit"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := hathi("", "doctor"); got.code != 1 || !strings.Contains(got.stdout, "not executable") {
		t.Errorf("got %d and %q, want 1", got.code, got.stdout)
	}
	gittest.Write(t, dir, ".hathi.yml", "no-ai-coauthor: maybe\n")
	if got := hathi("", "doctor"); got.code != 2 {
		t.Errorf("got %d and %q, want 2", got.code, got.stdout)
	}
	if got := hathi("", "list", "--format", "json"); got.code != 2 ||
		!strings.Contains(got.stdout, `{"problem":"no-ai-coauthor is not true or false","line":1}`) {
		t.Errorf("list: got %d and %q, want 2", got.code, got.stdout)
	}
}

func TestDoctorComparesTheVersionWithItsPin(t *testing.T) {
	for _, c := range []struct {
		mise, running string
		problem       bool
	}{
		{"", "1.0.0", false},
		{`"github:bunderlog/hathi" = "1.0"`, "1.0.3", false},
		{`"github:bunderlog/hathi" = "v1"`, "1.2.0", false},
		{`"github:bunderlog/hathi" = { version = "1.0.0" }`, "1.0.0", false},
		{`"github:bunderlog/hathi" = "latest"`, "0.9.0", false},
		{`"github:bunderlog/hathi" = "1.0"`, "devel", false},
		{`"github:bunderlog/hathi" = "1.1"`, "1.0.3", true},
		{`"github:bunderlog/hathi" = "1.0.1"`, "1.0.10", true},
	} {
		t.Run(c.mise+" "+c.running, func(t *testing.T) {
			dir := t.TempDir()
			if c.mise != "" {
				gittest.Write(t, dir, "mise.toml", "[tools]\n"+c.mise+"\n")
			}
			if d := versionDiagnosis(dir, c.running); d.problem != c.problem {
				t.Errorf("got %v, want problem %v", d, c.problem)
			}
		})
	}
	// mise.toml without the pin leaves it to .mise.toml.
	dir := t.TempDir()
	gittest.Write(t, dir, "mise.toml", "[tools]\ngo = \"1.27\"\n")
	gittest.Write(t, dir, ".mise.toml", "[tools]\n\"github:bunderlog/hathi\" = \"1.1\"\n")
	if d := versionDiagnosis(dir, "1.0.0"); !d.problem {
		t.Errorf("got %v, want the pin in .mise.toml to be a problem", d)
	}
}

// Criterion 12.
func TestVersion(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}} {
		t.Run(args[0], func(t *testing.T) {
			var stdout strings.Builder
			if code := Run("v1.0.0", args, nil, &stdout, &stdout); code != 0 || stdout.String() != "1.0.0\n" {
				t.Errorf("got %d and %q", code, stdout.String())
			}
		})
	}
}

func TestExplainsEachCheck(t *testing.T) {
	for _, name := range settings.Checks {
		t.Run(name, func(t *testing.T) {
			if got := hathi("", "explain", name); got.code != 0 || !strings.HasPrefix(got.stdout, name+" (") {
				t.Errorf("got %d and %q", got.code, got.stdout)
			}
		})
	}
}

func TestTheChecksAreTheSettingsKeys(t *testing.T) {
	var names []string
	for _, c := range all(settings.Default()) {
		names = append(names, c.Name)
	}
	if strings.Join(names, " ") != strings.Join(settings.Checks, " ") {
		t.Errorf("got %v, want %v", names, settings.Checks)
	}
}

// Criterion 9: no color codes, whatever the format.
func TestPrintsNoColor(t *testing.T) {
	repo(t)
	for _, f := range []string{"text", "json", "github"} {
		t.Run(f, func(t *testing.T) {
			got := hathi("Add x\n", "check", "commit-msg", "-", "--format", f)
			if got.code != 1 || strings.Contains(got.stdout+got.stderr, "\x1b") {
				t.Errorf("got %d and %q", got.code, got.stdout+got.stderr)
			}
		})
	}
}
