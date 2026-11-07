package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/bunderlog/hathi/internal/checks"
	"github.com/bunderlog/hathi/internal/checks/commit"
	"github.com/bunderlog/hathi/internal/settings"
)

// check is `hathi check`: in a Git hook, over a range, or over what a GitHub Actions event brings.
func check(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fallback := "text"
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		fallback = "github"
	}
	pos, values, err := flags(args, []string{"only", "skip", "format", "config", "range"},
		[]string{"only", "skip"})
	f := fallback
	if err == nil {
		formats := []string{fallback}
		for _, x := range []string{"text", "json", "github"} {
			if x != fallback {
				formats = append(formats, x)
			}
		}
		f, err = format(values, formats...)
		if err != nil {
			f = fallback
		}
	}
	r := &runner{out: output{f, stdout, stderr}, values: values}
	if err == nil {
		err = r.usage(pos)
	}
	if err != nil {
		r.fail(err)
		return r.code
	}
	switch {
	case len(pos) > 0 && pos[0] == "ci":
		return r.ci()
	case len(pos) > 0:
		return r.hook(pos[0], pos[1:], stdin)
	}
	return r.rangeOf(values["range"][0])
}

// A runner runs the Checks of one `hathi check` and keeps its exit code.
type runner struct {
	out    output
	values map[string][]string
	repo   checks.Repo
	code   int
	byBlob map[string]settings.Settings // each commit's Settings, by their file's id
	fixed  *settings.Settings           // the Settings of --config, once read
}

// usage says why the command line `pos` and the flags are not one `hathi check` takes.
func (r *runner) usage(pos []string) error {
	only, skip := r.values["only"], r.values["skip"]
	if len(only) > 0 && len(skip) > 0 {
		return usagef("--only and --skip don't go together")
	}
	hasRange := len(r.values["range"]) > 0
	hook := ""
	switch {
	case len(pos) == 0 && !hasRange:
		return usagef("hathi check takes a Git hook (%s), ci or --range <base>..<head>",
			strings.Join(Hooks, ", "))
	case len(pos) == 0:
	case hasRange:
		return usagef("--range doesn't go with %s", pos[0])
	case pos[0] == "ci":
		if len(pos) > 1 {
			return usagef("hathi check ci takes no arguments")
		}
		if os.Getenv("GITHUB_ACTIONS") != "true" {
			return usagef("hathi check ci runs in GitHub Actions; elsewhere, " +
				"hathi check --range <base>..<head>")
		}
	case pos[0] == "commit-msg" && len(pos) != 2:
		return usagef("usage: hathi check commit-msg <file>, or - for stdin")
	case pos[0] == "pre-commit" && len(pos) != 1:
		return usagef("hathi check pre-commit takes no arguments")
	case pos[0] == "pre-push" && len(pos) != 1 && len(pos) != 3:
		return usagef("usage: hathi check pre-push [<remote> <url>]")
	case !slices.Contains(Hooks, pos[0]):
		return usagef("%q is not a Git hook hathi checks: %s", pos[0], strings.Join(Hooks, ", "))
	default:
		hook = pos[0]
	}
	for _, name := range append(only, skip...) {
		switch h := hookOf(name); {
		case h == "":
			return usagef("%q is not a Check: %s", name, strings.Join(settings.Checks, ", "))
		case hook != "" && h != hook:
			return usagef("%s is a Check of %s, not %s", name, h, hook)
		}
	}
	return nil
}

func (r *runner) fail(err error) {
	r.out.error(err)
	r.code = 2
}

// selected are the Checks of the Git hook `hook` that run with the Settings `s` and the flags.
func (r *runner) selected(s settings.Settings, hook string) []checks.Check {
	var selected []checks.Check
	for _, c := range all(s) {
		on := s.On(c.Name)
		if only := r.values["only"]; len(only) > 0 {
			on = slices.Contains(only, c.Name)
		}
		if c.Hook == hook && on && !slices.Contains(r.values["skip"], c.Name) {
			selected = append(selected, c)
		}
	}
	return selected
}

// runs reports whether the Check `name` runs with the Settings `s` and the flags.
func (r *runner) runs(s settings.Settings, name string) bool {
	return slices.ContainsFunc(r.selected(s, hookOf(name)), func(c checks.Check) bool {
		return c.Name == name
	})
}

// run runs the Check `c` on `in` and prints its Verdicts, each with what `at` says of where.
func (r *runner) run(c checks.Check, in checks.Input, at Verdict) {
	found, err := c.Find(in)
	if err != nil {
		prefix := ""
		if at.Commit != "" {
			prefix = at.Commit + " "
		}
		r.fail(fmt.Errorf("%s%s: %w", prefix, c.Name, err))
		return
	}
	at.Check = c.Name
	if len(found) == 0 {
		r.out.verdict(at)
		return
	}
	for _, f := range found {
		v := at
		v.Fail, v.File, v.Line, v.Reason = true, f.File, f.Line, f.Reason
		if f.Commit != "" {
			v.Commit = f.Commit
		}
		r.out.verdict(v)
		r.code = max(r.code, 1)
	}
}

// hook runs the Checks of the Git hook `hook`, with the arguments Git gives it.
func (r *runner) hook(hook string, args []string, stdin io.Reader) int {
	top, err := toplevel()
	if err != nil {
		r.fail(err)
		return r.code
	}
	l, err := load(top, r.values)
	if err != nil {
		r.fail(err)
		return r.code
	}
	if len(l.problems) > 0 {
		for _, line := range l.problemLines() {
			r.fail(errors.New(line))
		}
		return r.code
	}
	var in checks.Input
	switch hook {
	case "commit-msg":
		var text []byte
		if args[0] == "-" {
			text, err = io.ReadAll(stdin)
		} else {
			text, err = os.ReadFile(args[0])
		}
		in.Message = string(text)
	case "pre-push":
		var text []byte
		text, err = io.ReadAll(stdin)
		in.Pushed = string(text)
	}
	if err != nil {
		r.fail(err)
		return r.code
	}
	for _, c := range r.selected(l.Settings, hook) {
		r.run(c, in, Verdict{})
	}
	return r.code
}

// rangeOf checks the range `arg`, `<base>..<head>`, from the merge base of the two.
func (r *runner) rangeOf(arg string) int {
	base, head, ok := strings.Cut(arg, "..")
	if !ok || base == "" || head == "" || strings.HasPrefix(head, ".") {
		r.fail(usagef("--range is <base>..<head>, not %s", arg))
		return r.code
	}
	if err := r.ready(base, head); err != nil {
		r.fail(err)
		return r.code
	}
	from, err := r.mergeBase(base, head)
	if err != nil {
		r.fail(err)
		return r.code
	}
	return r.commits(from, head, nil, nil)
}

// ready says why the repo can't check a range with the commits `ids`: it isn't one, its Git is
// too old to read a commit's own .gitattributes, or it lacks one of them.
func (r *runner) ready(ids ...string) error {
	if _, err := toplevel(); err != nil {
		return err
	}
	out, err := r.repo.Git("version")
	if err != nil {
		return err
	}
	if m := regexp.MustCompile(`(\d+)\.(\d+)`).FindStringSubmatch(out); m != nil {
		major, _ := strconv.Atoi(m[1])
		minor, _ := strconv.Atoi(m[2])
		if major < 2 || major == 2 && minor < 40 {
			return fmt.Errorf("this Git, %s.%s, is older than 2.40, which a range needs to check "+
				"each commit with its own .gitattributes", m[1], m[2])
		}
	}
	for _, id := range ids {
		if !r.has(id) {
			return fmt.Errorf("%s is not a commit this checkout has", id)
		}
	}
	return nil
}

func (r *runner) has(id string) bool {
	_, err := r.repo.Git("cat-file", "-e", id+"^{commit}")
	return err == nil
}

func (r *runner) mergeBase(a, b string) (string, error) {
	out, err := r.repo.Git("merge-base", a, b)
	if err != nil {
		return "", fmt.Errorf("%s and %s have no merge base: %w", a, b, err)
	}
	return strings.TrimSpace(out), nil
}

// commits checks each commit `head` has and `from` hasn't, "" for all, leaving out what `not` has,
// with the Verdicts its Git hooks would have given it, and the pull request's title where `title`
// isn't nil.
func (r *runner) commits(from, head string, not []string, title *string) int {
	args := []string{"rev-list", "--reverse", "--parents", head}
	if from != "" {
		args = append(args, "^"+from)
	}
	if len(not) > 0 {
		args = append(append(args, "--not"), not...)
	}
	list, err := r.repo.Git(args...)
	if err != nil {
		r.fail(err)
		return r.code
	}
	headID, err := r.repo.Git("rev-parse", head+"^{commit}")
	if err != nil {
		r.fail(err)
		return r.code
	}
	headID = strings.TrimSpace(headID)
	for line := range strings.Lines(list) {
		fields := strings.Fields(line)
		id, at := fields[0], Verdict{Commit: fields[0][:12]}
		s, err := r.settingsAt(id)
		if err != nil {
			return r.code
		}
		message, err := r.message(id)
		if err != nil {
			r.fail(err)
			continue
		}
		for _, c := range r.selected(s, "commit-msg") {
			r.run(c, checks.Input{Message: message}, at)
		}
		if len(fields) > 2 {
			// Its parents brought its changes; the pre-push Checks look at it below.
			continue
		}
		parent := ""
		if len(fields) > 1 {
			parent = fields[1]
		} else if parent, err = r.repo.EmptyTree(); err != nil {
			r.fail(err)
			continue
		}
		changes := checks.Changes{Repo: r.repo, Commit: id, Parent: parent}
		for _, c := range r.selected(s, "pre-commit") {
			r.run(c, checks.Input{Changes: changes}, at)
		}
	}
	s, err := r.settingsAt(headID)
	if err != nil {
		return r.code
	}
	whole := checks.Input{Repo: r.repo, Range: &checks.Range{Head: headID, From: from, Not: not}}
	for _, c := range r.selected(s, "pre-push") {
		r.run(c, whole, Verdict{})
	}
	if title != nil && s.PRTitle && r.runs(s, "conventional-commits") {
		r.run(commit.Check(s.Commit), checks.Input{Message: *title}, Verdict{PRTitle: true})
	}
	return r.code
}

// settingsAt are the Settings commit `id` is checked with: its own .hathi.yml, or the file
// --config names. With a Problem it prints each and returns an error.
func (r *runner) settingsAt(id string) (settings.Settings, error) {
	if len(r.values["config"]) > 0 {
		if r.fixed == nil {
			l, err := load("", r.values)
			if err == nil && len(l.problems) > 0 {
				err = errors.New(strings.Join(l.problemLines(), "; "))
			}
			if err != nil {
				r.fail(err)
				return settings.Settings{}, err
			}
			r.fixed = &l.Settings
		}
		return *r.fixed, nil
	}
	// Nothing where the commit has no such file, and an error where Git can't read the commit.
	blob, err := r.repo.Git("ls-tree", "--format=%(objectname)", id, "--", settings.File)
	if err != nil {
		r.fail(err)
		return settings.Settings{}, err
	}
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return settings.Default(), nil
	}
	if s, ok := r.byBlob[blob]; ok {
		return s, nil
	}
	text, err := r.repo.Git("cat-file", "blob", blob)
	if err != nil {
		r.fail(err)
		return settings.Settings{}, err
	}
	s, problems := settings.Parse(text)
	if len(problems) > 0 {
		l := loaded{s, id[:12] + ":" + settings.File, problems}
		for _, line := range l.problemLines() {
			r.fail(errors.New(line))
		}
		return s, errors.New("settings problems")
	}
	if r.byBlob == nil {
		r.byBlob = map[string]settings.Settings{}
	}
	r.byBlob[blob] = s
	return s, nil
}

// message is commit `id`'s message, as Git keeps it.
func (r *runner) message(id string) (string, error) {
	raw, err := r.repo.Git("cat-file", "commit", id)
	if err != nil {
		return "", err
	}
	_, message, _ := strings.Cut(raw, "\n\n")
	return message, nil
}

// An event is what of a GitHub Actions event `hathi check ci` reads.
type event struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Deleted    bool   `json:"deleted"`
	Repository struct {
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`
	PullRequest struct {
		Title string `json:"title"`
		Base  struct {
			SHA string `json:"sha"`
		} `json:"base"`
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	} `json:"pull_request"`
}

// ci checks what the GitHub Actions event brings (PRD, Command line).
func (r *runner) ci() int {
	name := os.Getenv("GITHUB_EVENT_NAME")
	if name != "pull_request" && name != "push" {
		r.fail(usagef("hathi check ci takes a pull_request or push event, not %q", name))
		return r.code
	}
	text, err := os.ReadFile(os.Getenv("GITHUB_EVENT_PATH"))
	var e event
	if err == nil {
		err = json.Unmarshal(text, &e)
	}
	if err != nil {
		r.fail(fmt.Errorf("the event: %w", err))
		return r.code
	}
	if _, err := toplevel(); err != nil {
		r.fail(err)
		return r.code
	}
	if name == "push" && (e.Deleted || !strings.HasPrefix(e.Ref, "refs/heads/") || checks.IsZero(e.After)) {
		return 0 // a tag or a deletion brings no commits
	}
	if out, err := r.repo.Git("rev-parse", "--is-shallow-repository"); err != nil ||
		strings.TrimSpace(out) == "true" {
		r.fail(errors.New("the checkout is shallow: give actions/checkout fetch-depth: 0"))
		return r.code
	}
	if name == "pull_request" {
		base, head := e.PullRequest.Base.SHA, e.PullRequest.Head.SHA
		if err := r.ready(base, head); err != nil {
			r.fail(err)
			return r.code
		}
		from, err := r.mergeBase(base, head)
		if err != nil {
			r.fail(err)
			return r.code
		}
		return r.commits(from, head, nil, &e.PullRequest.Title)
	}
	if err := r.ready(e.After); err != nil {
		r.fail(err)
		return r.code
	}
	branch, defaultBranch := strings.TrimPrefix(e.Ref, "refs/heads/"), e.Repository.DefaultBranch
	origin := "refs/remotes/origin/" + defaultBranch
	var not []string
	if branch != defaultBranch {
		// A branch rebased onto the default one isn't blamed for its commits and merges.
		not = []string{origin}
	}
	from := ""
	switch {
	case !checks.IsZero(e.Before) && r.has(e.Before):
		from, err = r.mergeBase(e.Before, e.After)
	case branch == defaultBranch && checks.IsZero(e.Before):
		// A first push of the default branch checks all its commits.
	case branch == defaultBranch:
		err = fmt.Errorf("the checkout lacks before, %s, which leaves no base to check from: "+
			"give actions/checkout fetch-depth: 0, though after a force push to the default branch "+
			"no history has it", e.Before)
	default:
		from, err = r.mergeBase(e.After, origin)
	}
	if err != nil {
		r.fail(err)
		return r.code
	}
	return r.commits(from, e.After, not, nil)
}
