package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/bunderlog/hathi/internal/settings"
)

// list is `hathi list`: every Check, its Git hook, whether it is on in this repo, and its settings.
func list(args []string, stdout, stderr io.Writer) int {
	pos, values, err := flags(args, []string{"format", "config"}, nil)
	f := "text"
	if err == nil && len(pos) > 0 {
		err = usagef("hathi list takes no arguments")
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
	out := output{f, stdout, stderr}
	for _, c := range all(l.Settings) {
		options := optionsOf(l.Settings, c.Name)
		if f == "json" {
			o := map[string]any{}
			for _, opt := range options {
				o[opt.key] = opt.value
			}
			out.json(struct {
				Check    string         `json:"check"`
				Hook     string         `json:"hook"`
				On       bool           `json:"on"`
				Settings map[string]any `json:"settings"`
			}{c.Name, c.Hook, l.On(c.Name), o})
			continue
		}
		on := "off"
		if l.On(c.Name) {
			on = "on"
		}
		var shown []string
		for _, opt := range options {
			shown = append(shown, opt.key+": "+opt.text())
		}
		line := fmt.Sprintf("%-22s %-11s %-3s %s", c.Name, c.Hook, on, strings.Join(shown, ", "))
		fmt.Fprintln(stdout, strings.TrimRight(line, " "))
	}
	for i, line := range l.problemLines() {
		if f == "json" {
			out.json(struct {
				Problem string `json:"problem"`
				Line    int    `json:"line"`
			}{l.problems[i].Message, l.problems[i].Line})
		} else {
			fmt.Fprintln(stdout, line)
		}
	}
	if len(l.problems) > 0 {
		return 2
	}
	return 0
}

// An option is a Check's setting and its value.
type option struct {
	key   string
	value any // a list, any, a number or a boolean, as JSON shows it
}

func (o option) text() string {
	if items, ok := o.value.([]string); ok {
		return "[" + strings.Join(items, ", ") + "]"
	}
	return fmt.Sprint(o.value)
}

// optionsOf are the settings of the Check `name` in `s`, in the order of settings.DefaultFile.
func optionsOf(s settings.Settings, name string) []option {
	listOrAny := func(items []string) any {
		if items == nil {
			return "any"
		}
		return items
	}
	switch name {
	case "conventional-commits":
		return []option{
			{"types", listOrAny(s.Commit.Types)}, {"scopes", listOrAny(s.Commit.Scopes)},
			{"max-length", s.Commit.MaxLength}, {"pr-title", s.PRTitle},
		}
	case "no-large-files":
		return []option{{"max-size", s.MaxSize}}
	}
	return nil
}
