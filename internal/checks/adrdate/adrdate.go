// Package adrdate is the Check no-stale-adr-date.
package adrdate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bunderlog/hathi/internal/checks"
)

// ADRs are where a project keeps its ADRs, one Markdown file per subject.
const ADRs = ".about/adr/"

// An ADR's Date line, ending in LF or CRLF.
var dateLine = regexp.MustCompile(`(?m)^Date: *(.*?) *\r?$`)

// Check is no-stale-adr-date: each ADR the changes edit without setting its Date to the day of the
// change, `today` (YYYY-MM-DD) for the staged changes and a commit's committer date for a commit's.
// A new ADR, a deleted or renamed one, and a file beside the ADRs pass.
func Check(today string) checks.Check {
	return checks.Check{
		Name:    "no-stale-adr-date",
		Hook:    "pre-commit",
		Explain: explanation,
		Find: func(in checks.Input) ([]checks.Finding, error) {
			listed, err := in.Changes.Diff(nil, "--no-renames", "--diff-filter=M", "--name-only", "-z")
			if err != nil {
				return nil, err
			}
			day, dated := today, in.Changes.Commit == ""
			var found []checks.Finding
			for path := range strings.SplitSeq(strings.TrimSuffix(listed, "\x00"), "\x00") {
				if !strings.HasPrefix(path, ADRs) || !strings.HasSuffix(path, ".md") {
					continue
				}
				if !dated {
					out, err := in.Changes.Git("show", "-s", "--format=%cs", in.Changes.Commit)
					if err != nil {
						return nil, err
					}
					day, dated = strings.TrimSpace(out), true
				}
				text, err := in.Changes.File(path)
				if err != nil {
					return nil, err
				}
				m := dateLine.FindStringSubmatchIndex(text)
				switch {
				case m == nil:
					found = append(found, checks.Finding{
						File:   path,
						Reason: fmt.Sprintf("has no Date line; add Date: %s under its title", day),
					})
				case text[m[2]:m[3]] != day:
					found = append(found, checks.Finding{
						File: path,
						Line: strings.Count(text[:m[0]], "\n") + 1,
						Reason: fmt.Sprintf("changed, but its Date is %s; set it to %s",
							text[m[2]:m[3]], day),
					})
				}
			}
			return found, nil
		},
	}
}

// explanation is what `hathi explain` prints for the Check: what it looks for, how to fix a
// failure, and how to mark a false alarm.
const explanation = `no-stale-adr-date (pre-commit) looks for an ADR in .about/adr/ that the
changes edit without setting its Date line to the day of the change: today for staged changes, the
commit's committer date for a commit in a range. A new ADR, a deleted or renamed one, and a file
beside the ADRs pass.

To fix it, set the ADR's line "Date: YYYY-MM-DD" under its title to that day, or add one.

There is no false alarm to mark: a project that keeps no dated ADRs turns the Check off with
no-stale-adr-date: false.
`
