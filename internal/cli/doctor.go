package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"go.yaml.in/yaml/v3"

	"github.com/bunderlog/hathi/internal/checks"
)

// A finding of `hathi doctor`: one thing it checked, and what it found.
type diagnosis struct {
	topic   string // settings, a Git hook's name, or version
	problem bool
	message string
}

// doctor is `hathi doctor`: what is wrong with how this repo runs `hathi`. It changes nothing.
func doctor(version string, args []string, stdout, stderr io.Writer) int {
	pos, values, err := flags(args, []string{"format", "config"}, nil)
	f := "text"
	if err == nil && len(pos) > 0 {
		err = usagef("hathi doctor takes no arguments")
	}
	if err == nil {
		f, err = format(values, "text", "json")
	}
	var top string
	if err == nil {
		top, err = toplevel()
	}
	var l loaded
	if err == nil {
		l, err = load(top, values)
	}
	if err != nil {
		fmt.Fprintln(stderr, "hathi: "+err.Error())
		return 2
	}
	var found []diagnosis
	switch {
	case len(l.problems) > 0:
		for _, line := range l.problemLines() {
			found = append(found, diagnosis{"settings", true, line})
		}
	case l.file == "":
		found = append(found, diagnosis{"settings", false,
			"no " + filepathOf(values) + ": every Check is on with its defaults"})
	default:
		found = append(found, diagnosis{"settings", false, l.file + " holds no problem"})
	}
	for _, hook := range Hooks {
		on := slices.ContainsFunc(all(l.Settings), func(c checks.Check) bool {
			return c.Hook == hook && l.On(c.Name)
		})
		if on {
			found = append(found, hookDiagnosis(top, hook))
		}
	}
	found = append(found, versionDiagnosis(top, versionOf(version)))
	code := 0
	out := output{f, stdout, stderr}
	for _, d := range found {
		if d.problem {
			code = 1
		}
		if f == "json" {
			out.json(struct {
				Topic   string `json:"topic"`
				Problem bool   `json:"problem"`
				Message string `json:"message"`
			}{d.topic, d.problem, d.message})
			continue
		}
		mark := "ok     "
		if d.problem {
			mark = "problem"
		}
		fmt.Fprintf(stdout, "%s  %s: %s\n", mark, d.topic, d.message)
	}
	if len(l.problems) > 0 {
		return 2
	}
	return code
}

func filepathOf(values map[string][]string) string {
	if len(values["config"]) > 0 {
		return values["config"][0]
	}
	return ".hathi.yml"
}

// hookDiagnosis says whether Git runs `hathi check <hook>` in the repo at `top`: a line with it
// in the hook Git runs, or in the husky or lefthook file that hook hands over to.
func hookDiagnosis(top, hook string) diagnosis {
	line := "hathi check " + hook
	dir, err := checks.Repo{Dir: top}.Git("rev-parse", "--git-path", "hooks")
	if err != nil {
		return diagnosis{hook, true, err.Error()}
	}
	path := strings.TrimSpace(dir)
	if !filepath.IsAbs(path) {
		path = filepath.Join(top, path)
	}
	path = filepath.Join(path, hook)
	shown, _ := filepath.Rel(top, path)
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return diagnosis{hook, true, fmt.Sprintf(
			"Git runs no %s hook (%s), so its Checks don't run here; hathi hooks print %s prints one",
			hook, shown, hook)}
	}
	if err != nil {
		return diagnosis{hook, true, err.Error()}
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		return diagnosis{hook, true, shown + " is not executable, so Git doesn't run it: chmod +x " + shown}
	}
	if text, err := os.ReadFile(path); err == nil && strings.Contains(string(text), line) {
		return diagnosis{hook, false, shown + " runs " + line}
	}
	if text, err := os.ReadFile(filepath.Join(top, ".husky", hook)); err == nil &&
		strings.Contains(string(text), line) {
		return diagnosis{hook, false, ".husky/" + hook + " runs " + line}
	}
	for _, name := range []string{"lefthook.yml", ".lefthook.yml"} {
		if lefthookRuns(filepath.Join(top, name), hook, line) {
			return diagnosis{hook, false, name + " runs " + line + " in " + hook}
		}
	}
	return diagnosis{hook, true, shown + " doesn't run " + line + ", so its Checks don't run here"}
}

// lefthookRuns reports whether the lefthook file at `path` has `line` in what it runs for `hook`.
func lefthookRuns(path, hook, line string) bool {
	text, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var doc yaml.Node
	if yaml.Unmarshal(text, &doc) != nil || len(doc.Content) == 0 {
		return false
	}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == hook {
			return holds(root.Content[i+1], line)
		}
	}
	return false
}

func holds(n *yaml.Node, line string) bool {
	if n.Kind == yaml.ScalarNode {
		return strings.Contains(n.Value, line)
	}
	return slices.ContainsFunc(n.Content, func(c *yaml.Node) bool { return holds(c, line) })
}

// The tool mise installs `hathi` as (ADR distribution).
const miseTool = "github:bunderlog/hathi"

// versionDiagnosis says whether `running`, this binary's version, is the one the repo at `top`
// pins in mise.toml, or else in .mise.toml.
func versionDiagnosis(top, running string) diagnosis {
	file, pin := "", ""
	for _, name := range []string{"mise.toml", ".mise.toml"} {
		text, err := os.ReadFile(filepath.Join(top, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		var m struct {
			Tools map[string]any `toml:"tools"`
		}
		if err == nil {
			_, err = toml.Decode(string(text), &m)
		}
		if err != nil {
			return diagnosis{"version", true, name + ": " + err.Error()}
		}
		file = name
		switch v := m.Tools[miseTool].(type) {
		case string:
			pin = v
		case map[string]any:
			pin, _ = v["version"].(string)
		}
		if pin != "" {
			break
		}
	}
	switch {
	case pin == "" && file == "":
		return diagnosis{"version", false, "running " + running + "; no mise.toml pins " + miseTool}
	case pin == "":
		return diagnosis{"version", false, "running " + running + "; " + file + " pins no " + miseTool}
	case pin == "latest":
		return diagnosis{"version", false, "running " + running + "; " + file + " pins latest"}
	case running == "devel":
		return diagnosis{"version", false, "running a build with no version; " + file + " pins " + pin}
	case !matches(strings.TrimPrefix(pin, "v"), running):
		return diagnosis{"version", true, fmt.Sprintf(
			"running %s, but %s pins %s: mise install installs it", running, file, pin)}
	}
	return diagnosis{"version", false, "running " + running + ", as " + file + " pins " + pin}
}

// matches reports whether `pin` is `version` or its start by whole parts: 1.0 matches 1.0.3.
func matches(pin, version string) bool {
	p, v := strings.Split(pin, "."), strings.Split(version, ".")
	return len(p) <= len(v) && slices.Equal(p, v[:len(p)])
}
