package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// A Verdict is what a Check returns, one per finding: a failing one, or a passing one for an input
// it finds nothing in (glossary).
type Verdict struct {
	Check   string
	Fail    bool
	Commit  string // 12 characters; "" outside a range
	PRTitle bool   // for the pull request's title, which has no commit
	File    string // "" for none
	Line    int    // 0 for none
	Reason  string // "" for a passing one
}

// An output prints Verdicts and errors in one of the formats of `hathi check`: text on stderr,
// json or github on stdout.
type output struct {
	format         string
	stdout, stderr io.Writer
}

func (o output) verdict(v Verdict) {
	switch o.format {
	case "text":
		if !v.Fail {
			return
		}
		where := ""
		if v.PRTitle {
			where = "PR title "
		} else if v.Commit != "" {
			where = v.Commit + " "
		}
		where += v.Check + ": "
		if v.File != "" {
			where += v.File + ":"
		}
		if v.Line > 0 {
			where += fmt.Sprintf("%d:", v.Line)
		}
		if v.File != "" || v.Line > 0 {
			where += " "
		}
		fmt.Fprintf(o.stderr, "%s%s; run: hathi explain %s\n", where, v.Reason, v.Check)
	case "json":
		verdict := "pass"
		if v.Fail {
			verdict = "fail"
		}
		o.json(struct {
			Check   string `json:"check"`
			Verdict string `json:"verdict"`
			Reason  string `json:"reason"`
			Commit  string `json:"commit,omitempty"`
			File    string `json:"file,omitempty"`
			Line    int    `json:"line,omitempty"`
		}{v.Check, verdict, v.Reason, v.Commit, v.File, v.Line})
	case "github":
		if !v.Fail {
			return
		}
		var props []string
		title := v.Check
		if v.File != "" {
			props = append(props, "file="+escapeProperty(v.File))
			if v.Line > 0 {
				props = append(props, fmt.Sprintf("line=%d", v.Line))
			}
		} else if v.PRTitle {
			title += " PR title"
		} else if v.Commit != "" {
			title += " " + v.Commit
		}
		props = append(props, "title="+escapeProperty(title))
		fmt.Fprintf(o.stdout, "::error %s::%s\n", strings.Join(props, ","), escapeData(v.Reason))
	}
}

// error prints why `hathi` can't run or can't check: one line on stderr, and in github an
// annotation too.
func (o output) error(err error) {
	fmt.Fprintln(o.stderr, "hathi: "+err.Error())
	if o.format == "github" {
		fmt.Fprintf(o.stdout, "::error title=hathi::%s\n", escapeData(err.Error()))
	}
}

// json prints `v` as one line of JSON on stdout.
func (o output) json(v any) {
	e := json.NewEncoder(o.stdout)
	e.SetEscapeHTML(false)
	_ = e.Encode(v)
}

// escapeData escapes a workflow command's message as GitHub requires.
func escapeData(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(s)
}

// escapeProperty escapes a workflow command's property as GitHub requires.
func escapeProperty(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C").Replace(s)
}
