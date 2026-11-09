// Package cli is `hathi`'s command line (PRD, Command line).
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/bunderlog/hathi/internal/checks"
	"github.com/bunderlog/hathi/internal/checks/adrdate"
	"github.com/bunderlog/hathi/internal/checks/coauthor"
	"github.com/bunderlog/hathi/internal/checks/commit"
	"github.com/bunderlog/hathi/internal/checks/conflict"
	"github.com/bunderlog/hathi/internal/checks/history"
	"github.com/bunderlog/hathi/internal/checks/secrets"
	"github.com/bunderlog/hathi/internal/checks/size"
	"github.com/bunderlog/hathi/internal/settings"
)

// Hooks are the Git hooks that run Checks, in the order Git runs them.
var Hooks = []string{"commit-msg", "pre-commit", "pre-push"}

// all are the Checks with the Settings `s`, in the order of settings.Checks.
func all(s settings.Settings) []checks.Check {
	return []checks.Check{
		commit.Check(s.Commit), coauthor.Check, secrets.Check, conflict.Check, size.Check(s.MaxSize),
		adrdate.Check(time.Now().Format(time.DateOnly)), history.Check,
	}
}

// hookOf is the Git hook of the Check `name`, "" for no such Check.
func hookOf(name string) string {
	for _, c := range all(settings.Default()) {
		if c.Name == name {
			return c.Hook
		}
	}
	return ""
}

// A usageError is a command line `hathi` can't run; it exits 2.
type usageError string

func (e usageError) Error() string { return string(e) }

func usagef(format string, a ...any) error { return usageError(fmt.Sprintf(format, a...)) }

// Run runs `hathi` with the arguments `args` and returns its exit code. `version` is the one the
// build was given, "" for none.
func Run(version string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fail := func(err error) int {
		fmt.Fprintln(stderr, "hathi: "+err.Error())
		return 2
	}
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	command, rest := args[0], args[1:]
	switch command {
	case "version", "--version":
		if len(rest) > 0 {
			return fail(usagef("hathi version takes no arguments"))
		}
		fmt.Fprintln(stdout, versionOf(version))
		return 0
	case "help", "-h", "--help":
		return help(rest, stdout, stderr)
	case "explain":
		return explain(rest, stdout, stderr)
	case "hooks":
		if len(rest) != 2 || rest[0] != "print" || !slices.Contains(Hooks, rest[1]) {
			return fail(usagef("usage: hathi hooks print <%s>", strings.Join(Hooks, "|")))
		}
		fmt.Fprintf(stdout, "#!/bin/sh\nexec hathi check %s \"$@\"\n", rest[1])
		return 0
	case "config":
		if len(rest) != 1 || rest[0] != "--default" {
			return fail(usagef("usage: hathi config --default"))
		}
		fmt.Fprint(stdout, settings.DefaultFile)
		return 0
	case "check":
		return check(rest, stdin, stdout, stderr)
	case "list":
		return list(rest, stdout, stderr)
	case "doctor":
		return doctor(version, rest, stdout, stderr)
	}
	return fail(usagef("%q is not a command; hathi help lists them", command))
}

// A Release's version, as a tag names it; a pseudo-version Go gives a build from a checkout is not.
var released = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`)

// versionOf is the binary's own version: the one the build was given, or the module's when
// `go install …@v<version>` built it, or devel for neither.
func versionOf(given string) string {
	if given != "" {
		return strings.TrimPrefix(given, "v")
	}
	if info, ok := debug.ReadBuildInfo(); ok && released.MatchString(info.Main.Version) {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "devel"
}

// flags splits `args` into the positional ones and the values of the flags `names` takes, as
// `--name value` or `--name=value`; the ones in `many` may be repeated.
func flags(args []string, names, many []string) (pos []string, values map[string][]string, err error) {
	values = map[string][]string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-" || !strings.HasPrefix(a, "-") {
			pos = append(pos, a)
			continue
		}
		name, value, hasValue := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		if !strings.HasPrefix(a, "--") || !slices.Contains(names, name) {
			return nil, nil, usagef("%s is not a flag here; hathi help lists them", a)
		}
		if !hasValue {
			if i+1 == len(args) {
				return nil, nil, usagef("--%s takes a value", name)
			}
			i++
			value = args[i]
		}
		if len(values[name]) > 0 && !slices.Contains(many, name) {
			return nil, nil, usagef("--%s is given twice", name)
		}
		values[name] = append(values[name], value)
	}
	return pos, values, nil
}

// format is the value of --format, one of `formats`, the first by default.
func format(values map[string][]string, formats ...string) (string, error) {
	if len(values["format"]) == 0 {
		return formats[0], nil
	}
	f := values["format"][0]
	if !slices.Contains(formats, f) {
		return "", usagef("--format is %s", strings.Join(formats, ", "))
	}
	return f, nil
}

// toplevel is the root of the repo the current folder is in.
func toplevel() (string, error) {
	out, err := checks.Repo{}.Git("rev-parse", "--show-toplevel")
	if err != nil {
		return "", errors.New("not in a Git repo")
	}
	return strings.TrimSpace(out), nil
}

// A loaded is the Settings read for a command, where they came from, and their Problems.
type loaded struct {
	settings.Settings
	file     string // as the user knows it: the path --config gave, or .hathi.yml; "" for none
	problems []settings.Problem
}

// load reads the Settings of the repo at `top`, or of the file --config names in `values`.
func load(top string, values map[string][]string) (loaded, error) {
	path, file := filepath.Join(top, settings.File), settings.File
	if len(values["config"]) > 0 {
		path, file = values["config"][0], values["config"][0]
	}
	text, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && len(values["config"]) == 0 {
		return loaded{Settings: settings.Default()}, nil
	}
	if err != nil {
		return loaded{}, err
	}
	s, problems := settings.Parse(string(text))
	return loaded{s, file, problems}, nil
}

// problemLines are the Problems of `l`, one line each.
func (l loaded) problemLines() []string {
	var lines []string
	for _, p := range l.problems {
		if p.Line > 0 {
			lines = append(lines, fmt.Sprintf("%s:%d: %s", l.file, p.Line, p.Message))
		} else {
			lines = append(lines, l.file+": "+p.Message)
		}
	}
	return lines
}
