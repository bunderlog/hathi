// Package size is the Check no-large-files.
package size

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bunderlog/hathi/internal/checks"
)

// DefaultMaxSize is the largest file a commit may add without a setting, in KB.
const DefaultMaxSize = 1024

// Check is no-large-files: each file the changes add over `maxSize` KB, of 1024 bytes. A file
// already committed passes, changed or renamed too, and so does a submodule.
func Check(maxSize int) checks.Check {
	return checks.Check{
		Name:    "no-large-files",
		Hook:    "pre-commit",
		Explain: explanation,
		Find: func(in checks.Input) ([]checks.Finding, error) {
			// Each file added is `:<old mode> <new mode> <old id> <new id> A`, then its path.
			raw, err := in.Changes.Diff(nil, "-M", "--diff-filter=A", "--raw", "-z", "--no-abbrev")
			if err != nil {
				return nil, err
			}
			var paths []string
			var ids strings.Builder
			fields := strings.Split(strings.TrimSuffix(raw, "\x00"), "\x00")
			for i := 0; i+1 < len(fields); i += 2 {
				f := strings.Fields(fields[i])
				if len(f) != 5 || f[1] == "160000" {
					continue
				}
				paths = append(paths, fields[i+1])
				ids.WriteString(f[3])
				ids.WriteString("\n")
			}
			if len(paths) == 0 {
				return nil, nil
			}
			sizes, err := in.Changes.GitOK(nil, ids.String(), "cat-file", "--batch-check=%(objectsize)")
			if err != nil {
				return nil, err
			}
			var found []checks.Finding
			for i, line := range checks.Lines(sizes) {
				size, err := strconv.Atoi(line)
				if err != nil || i >= len(paths) {
					return nil, fmt.Errorf("git cat-file: \"%s\" is not a size", line)
				}
				if size > maxSize*1024 {
					found = append(found, checks.Finding{
						File:   paths[i],
						Reason: fmt.Sprintf("%d KB, over %d KB", (size+1023)/1024, maxSize),
					})
				}
			}
			return found, nil
		},
	}
}

// explanation is what `hathi explain` prints for the Check: what it looks for, how to fix a
// failure, and how to mark a false alarm.
const explanation = `no-large-files (pre-commit) looks for a file the changes add over max-size
KB, 1024 by default; a file already committed passes, changed or renamed too, and so does a
submodule.

To fix it, keep the file out of Git (git rm --cached <file>, then .gitignore it), or keep it in Git
LFS or a release asset.

To let it in, raise max-size under no-large-files in .hathi.yml.
`
