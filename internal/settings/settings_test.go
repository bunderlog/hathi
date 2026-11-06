package settings

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/bunderlog/hathi/internal/checks/commit"
)

// Criterion 7: with no file, or an empty one, every Check is on with its defaults, and
// `hathi config --default` prints exactly those defaults.
func TestDefaultsWithoutSettings(t *testing.T) {
	want := Settings{
		Commit: commit.Rules{
			Types: []string{"build", "chore", "ci", "docs", "feat", "fix", "perf", "refactor",
				"revert", "style", "test"},
			MaxLength: 100,
		},
		MaxSize: 1024,
	}
	for _, text := range []string{"", "# nothing\n", "~\n", DefaultFile} {
		t.Run(fmt.Sprintf("%.20q", text), func(t *testing.T) {
			got, problems := Parse(text)
			if !reflect.DeepEqual(got, want) || problems != nil {
				t.Errorf("got %+v and %v, want %+v", got, problems, want)
			}
			for _, name := range Checks {
				if !got.On(name) {
					t.Errorf("%s is off", name)
				}
			}
		})
	}
}

func TestTakesEachSetting(t *testing.T) {
	got, problems := Parse(`
conventional-commits:
  types: any
  scopes: [api, ui]
  max-length: 0
  pr-title: true
no-ai-coauthor: false
no-large-files: {max-size: 5000}
linear-history: false
`)
	want := Settings{
		Off:     []string{"no-ai-coauthor", "linear-history"},
		Commit:  commit.Rules{Scopes: []string{"api", "ui"}},
		PRTitle: true,
		MaxSize: 5000,
	}
	if !reflect.DeepEqual(got, want) || problems != nil {
		t.Errorf("got %+v and %v, want %+v", got, problems, want)
	}
}

// Criterion 7: an unknown key or a bad value is a Problem with its line; the other settings
// still apply.
func TestTellsEachProblemByItsLine(t *testing.T) {
	got, problems := Parse(`no-stale-adr-date: true
no-secrets: true
conventional-commits:
  types: [feat, 1]
  scopes: []
  max-length: -1
  pr-title: yes
  body: true
no-large-files: {max-size: 0}
no-ai-coauthor: off
linear-history: false
`)
	want := []Problem{
		{1, `"no-stale-adr-date" is not a setting of hathi: it stays in baloo`},
		{2, `"no-secrets" is not a setting`},
		{4, "conventional-commits: types is not a list of strings or any"},
		{5, "conventional-commits: scopes holds none: use any for any"},
		{6, "conventional-commits: max-length is -1, under 0"},
		{7, "conventional-commits: pr-title is not true or false"},
		{8, `"body" is not a setting of conventional-commits`},
		{9, "no-large-files: max-size is 0, under 1"},
		{10, "no-ai-coauthor is not true or false"},
	}
	if !slices.Equal(problems, want) {
		t.Errorf("got %v, want %v", problems, want)
	}
	if got.On("linear-history") || !reflect.DeepEqual(got.Commit, commit.Default) || got.MaxSize != 1024 {
		t.Errorf("got %+v, want linear-history off and the rest at its defaults", got)
	}
}

func TestTellsAFileThatIsNoYAML(t *testing.T) {
	_, problems := Parse("linear-history: true\nno-ai-coauthor: [\n")
	if len(problems) != 1 || problems[0].Line == 0 {
		t.Errorf("got %v, want one Problem with its line", problems)
	}
	_, problems = Parse("- a\n")
	if len(problems) != 1 || problems[0].Line != 1 {
		t.Errorf("got %v, want one Problem on line 1", problems)
	}
}

// The JSON Schema editors complete the file with has the keys Parse takes, and no other.
func TestSchemaHasTheSettingsKeys(t *testing.T) {
	text, err := os.ReadFile("../../schema.json")
	if err != nil {
		t.Fatal(err)
	}
	type object struct {
		Properties map[string]struct {
			OneOf []object `json:"oneOf"`
		} `json:"properties"`
	}
	var schema object
	if err := json.Unmarshal(text, &schema); err != nil {
		t.Fatal(err)
	}
	keys := func(o object) []string { return slices.Sorted(maps.Keys(o.Properties)) }
	if got, want := keys(schema), slices.Sorted(slices.Values(Checks)); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	for name, want := range map[string][]string{
		"conventional-commits": {"max-length", "pr-title", "scopes", "types"},
		"no-large-files":       {"max-size"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := keys(schema.Properties[name].OneOf[1]); !slices.Equal(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}
