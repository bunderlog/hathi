package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/bunderlog/hathi/internal/settings"
)

// explain is `hathi explain <check>`.
func explain(args []string, stdout, stderr io.Writer) int {
	for _, c := range all(settings.Default()) {
		if len(args) == 1 && args[0] == c.Name {
			fmt.Fprint(stdout, c.Explain)
			return 0
		}
	}
	fmt.Fprintf(stderr, "hathi: usage: hathi explain <%s>\n", strings.Join(settings.Checks, "|"))
	return 2
}
