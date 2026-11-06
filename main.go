// Command hathi runs the commit and repo Checks in Git hooks and in CI. It sits at the module's
// root, not in cmd/hathi, so `go install github.com/bunderlog/hathi@v<version>` installs it under
// its own name.
package main

import (
	"os"

	"github.com/bunderlog/hathi/internal/cli"
)

// version is the Release's, which its build sets; "" for none.
var version string

func main() {
	os.Exit(cli.Run(version, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
