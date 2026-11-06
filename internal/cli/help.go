package cli

import (
	"fmt"
	"io"
	"strings"
)

// The usage of each command, in the order `hathi help` lists them.
var commands = []struct{ name, usage string }{
	{"check", `hathi check commit-msg <file>   checks the message in <file>, or on stdin for -
hathi check pre-commit          checks the staged changes
hathi check pre-push            checks the pushed refs on stdin, as Git gives them
hathi check ci                  checks what a GitHub Actions push or pull_request brings
hathi check --range <base>..<head>
                                checks each commit of the range from their merge base
  --only <check>, --skip <check>   run or leave out Checks; each may be repeated
  --format text|json|github        github by default in GitHub Actions, else text
  --config <path>                  the settings' file, .hathi.yml at the root by default
`},
	{"list", `hathi list [--format text|json] [--config <path>]
                                every Check, its Git hook, on or off here, and its settings
`},
	{"explain", `hathi explain <check>           what a Check looks for, and how to fix a failure
`},
	{"doctor", `hathi doctor [--format text|json] [--config <path>]
                                the settings' problems, the Git hooks that don't run hathi, and
                                whether this is the version mise.toml pins
`},
	{"hooks", `hathi hooks print <hook>        prints the hook script for commit-msg, pre-commit or
                                pre-push; writes no file
`},
	{"config", `hathi config --default          prints the default settings with every key
`},
	{"version", `hathi version                   prints its own version
`},
	{"help", `hathi help [<command>]          prints the usage of hathi or of one command
`},
}

var usage = func() string {
	var b strings.Builder
	b.WriteString("hathi: commit and repo Checks for Git hooks and CI\n\n")
	for _, c := range commands {
		b.WriteString(c.usage)
	}
	return b.String()
}()

// help is `hathi help [<command>]`.
func help(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return 0
	}
	for _, c := range commands {
		if len(args) == 1 && args[0] == c.name {
			fmt.Fprint(stdout, c.usage)
			return 0
		}
	}
	fmt.Fprintf(stderr, "hathi: %q is not a command; hathi help lists them\n", strings.Join(args, " "))
	return 2
}
