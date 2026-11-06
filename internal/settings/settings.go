// Package settings reads the Settings, `.hathi.yml` (ADR config).
package settings

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/bunderlog/hathi/internal/checks/commit"
	"github.com/bunderlog/hathi/internal/checks/size"
)

// File is the Settings' file at the repo's root.
const File = ".hathi.yml"

// Checks are the Checks' names, each a key of the Settings, in the order the PRD lists them.
var Checks = []string{
	"conventional-commits", "no-ai-coauthor", "no-secrets-in-commits", "no-conflict-markers",
	"no-large-files", "linear-history",
}

// Settings are what a repo sets for `hathi`.
type Settings struct {
	Off     []string     // the Checks turned off; every other is on
	Commit  commit.Rules // conventional-commits'
	PRTitle bool         // whether `hathi check ci` checks a pull request's title too
	MaxSize int          // no-large-files', in KB
}

// Default are the Settings without a file, or of an empty one.
func Default() Settings {
	return Settings{Commit: commit.Default, MaxSize: size.DefaultMaxSize}
}

// On reports whether the Settings turn the Check `name` on.
func (s Settings) On(name string) bool {
	return !slices.Contains(s.Off, name)
}

// A Problem is what the Settings' text has that hathi can't take: text that isn't YAML or isn't a
// mapping, a key that isn't a setting, or a value that doesn't decode (glossary).
type Problem struct {
	Line    int // 1-based; 0 where the file says none
	Message string
}

var yamlLine = regexp.MustCompile(`^yaml: line (\d+): (.*)$`)

// Parse reads the Settings in `text`, the file's. Each setting with a Problem keeps its default,
// and every other applies.
func Parse(text string) (Settings, []Problem) {
	s := Default()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil {
		if m := yamlLine.FindStringSubmatch(err.Error()); m != nil {
			line, _ := strconv.Atoi(m[1])
			return s, []Problem{{line, m[2]}}
		}
		return s, []Problem{{0, strings.TrimPrefix(err.Error(), "yaml: ")}}
	}
	if len(doc.Content) == 0 || isNull(doc.Content[0]) {
		return s, nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return s, []Problem{{root.Line, "the settings are not a mapping of each Check to its own"}}
	}
	var problems []Problem
	problem := func(n *yaml.Node, format string, a ...any) {
		problems = append(problems, Problem{n.Line, fmt.Sprintf(format, a...)})
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, value := root.Content[i], root.Content[i+1]
		name := key.Value
		switch {
		case name == "no-stale-adr-date":
			problem(key, "%q is not a setting of hathi: it stays in baloo", name)
			continue
		case !slices.Contains(Checks, name):
			problem(key, "%q is not a setting", name)
			continue
		}
		if on, ok := boolean(value); ok {
			if !on {
				s.Off = append(s.Off, name)
			}
			continue
		}
		options := map[string]func(*yaml.Node) string{}
		switch name {
		case "conventional-commits":
			options = map[string]func(*yaml.Node) string{
				"types": func(n *yaml.Node) string {
					return list(n, &s.Commit.Types, false)
				},
				"scopes": func(n *yaml.Node) string {
					return list(n, &s.Commit.Scopes, true)
				},
				"max-length": func(n *yaml.Node) string {
					return integer(n, &s.Commit.MaxLength, 0)
				},
				"pr-title": func(n *yaml.Node) string {
					on, ok := boolean(n)
					if !ok {
						return "is not true or false"
					}
					s.PRTitle = on
					return ""
				},
			}
		case "no-large-files":
			options = map[string]func(*yaml.Node) string{
				"max-size": func(n *yaml.Node) string { return integer(n, &s.MaxSize, 1) },
			}
		}
		if len(options) == 0 || value.Kind != yaml.MappingNode {
			if len(options) == 0 {
				problem(value, "%s is not true or false", name)
			} else {
				problem(value, "%s is not true, false or its settings", name)
			}
			continue
		}
		for j := 0; j+1 < len(value.Content); j += 2 {
			k, v := value.Content[j], value.Content[j+1]
			decode, ok := options[k.Value]
			if !ok {
				problem(k, "%q is not a setting of %s", k.Value, name)
			} else if why := decode(v); why != "" {
				problem(v, "%s: %s %s", name, k.Value, why)
			}
		}
	}
	return s, problems
}

func isNull(n *yaml.Node) bool {
	return n.Kind == yaml.ScalarNode && n.Tag == "!!null"
}

// boolean decodes `n` as YAML reads a boolean, True and FALSE too.
func boolean(n *yaml.Node) (on, ok bool) {
	if n.Kind != yaml.ScalarNode || n.Tag != "!!bool" || n.Decode(&on) != nil {
		return false, false
	}
	return on, true
}

// list decodes `n`, a list of strings or `any`, into `to`, nil for any. A list holds at least one
// where `nonEmpty`.
func list(n *yaml.Node, to *[]string, nonEmpty bool) string {
	if n.Kind == yaml.ScalarNode && n.Tag == "!!str" && n.Value == "any" {
		*to = nil
		return ""
	}
	why := "is not a list of strings or any"
	if n.Kind != yaml.SequenceNode {
		return why
	}
	items := []string{}
	for _, item := range n.Content {
		if item.Kind != yaml.ScalarNode || item.Tag != "!!str" {
			return why
		}
		items = append(items, item.Value)
	}
	if nonEmpty && len(items) == 0 {
		return "holds none: use any for any"
	}
	*to = items
	return ""
}

// integer decodes `n`, a whole number of at least `least`, into `to`.
func integer(n *yaml.Node, to *int, least int) string {
	i, err := strconv.Atoi(n.Value)
	if n.Kind != yaml.ScalarNode || n.Tag != "!!int" || err != nil {
		return "is not a whole number"
	}
	if i < least {
		return fmt.Sprintf("is %d, under %d", i, least)
	}
	*to = i
	return ""
}

// Schema is where the JSON Schema of the Settings is published (ADR config).
const Schema = "https://raw.githubusercontent.com/bunderlog/hathi/main/schema.json"

// DefaultFile is the Settings' file with every key at its default, and a comment for each.
const DefaultFile = `# yaml-language-server: $schema=` + Schema + `
#
# hathi's settings for this repo: each Check on with true, off with false, or its settings. A
# Check without its key is on with its defaults, the ones below.

# commit-msg: the message is a Conventional Commit, as commitlint's config-conventional reads it.
conventional-commits:
  # The types a subject may have, or any.
  types: [build, chore, ci, docs, feat, fix, perf, refactor, revert, style, test]
  # The scopes a subject may have, or any; a subject without a scope passes either way.
  scopes: any
  # The longest line, in characters; 0 for no limit.
  max-length: 100
  # Whether hathi check ci checks a pull request's title too: on where the repo allows squash
  # merges, since GitHub takes a squashed commit's message from the title.
  pr-title: false

# commit-msg: no AI as a commit's co-author, and no line crediting one; a person's passes.
no-ai-coauthor: true

# pre-commit: the changes add no Secret and no Env file; hathi:allow-secret marks a false alarm.
no-secrets-in-commits: true

# pre-commit: the changes add no conflict marker; a longer conflict-marker-size in
# .gitattributes lets a file hold them.
no-conflict-markers: true

# pre-commit: the changes add no file over max-size; a file already committed passes.
no-large-files:
  # The largest file a commit may add, in KB of 1024 bytes.
  max-size: 1024

# pre-push: the push sends no merge commit.
linear-history: true
`
