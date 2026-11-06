// Package conflict is the Check no-conflict-markers.
package conflict

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/bunderlog/hathi/internal/checks"
)

// Ends a line of `git diff --check` for a conflict marker.
const leftover = ": leftover conflict marker"

var number = regexp.MustCompile(`^[+-]?[0-9]+$`)

// Check is no-conflict-markers: each conflict marker the changes add, as Git's own `diff --check`
// finds them, in a file where one of them opens or closes a conflict: a Markdown heading
// underlined with `=======` alone passes, and so does a file whose conflict-marker-size in
// .gitattributes is longer.
var Check = checks.Check{
	Name: "no-conflict-markers",
	Hook: "pre-commit",
	Find: func(in checks.Input) ([]checks.Finding, error) {
		// It exits 2 where it finds a marker or a whitespace error.
		out, err := in.Changes.Diff([]int{2}, "--check", "--no-color")
		if err != nil {
			return nil, err
		}
		var paths []string
		lines := map[string][]int{}
		for text := range strings.SplitSeq(out, "\n") {
			if !strings.HasSuffix(text, leftover) {
				continue
			}
			path, n, ok := cutLast(strings.TrimSuffix(text, leftover), ":")
			if !ok || !number.MatchString(n) {
				continue
			}
			if _, seen := lines[path]; !seen {
				paths = append(paths, path)
			}
			i, _ := strconv.Atoi(n)
			lines[path] = append(lines[path], i)
		}
		var found []checks.Finding
		for _, path := range paths {
			if opensConflict(in.Changes, path, lines[path]) {
				for _, n := range lines[path] {
					found = append(found, checks.Finding{File: path, Line: n, Reason: "a conflict marker"})
				}
			}
		}
		return found, nil
	},
}

// opensConflict reports whether one of the lines `lines` of the file `path` the changes leave
// opens or closes a conflict, `<<<<<<<` or `>>>>>>>`; it says so too where the file can't be
// read, such as a path Git quoted.
func opensConflict(changes checks.Changes, path string, lines []int) bool {
	text, err := changes.File(path)
	if err != nil {
		return true
	}
	all := strings.Split(text, "\n")
	for _, n := range lines {
		if n >= 1 && n <= len(all) && (strings.HasPrefix(all[n-1], "<") || strings.HasPrefix(all[n-1], ">")) {
			return true
		}
	}
	return false
}

// cutLast is strings.Cut at the last `sep`.
func cutLast(s, sep string) (before, after string, found bool) {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return "", s, false
	}
	return s[:i], s[i+len(sep):], true
}
