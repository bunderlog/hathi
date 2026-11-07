package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/bunderlog/hathi/internal/settings"
)

// explanations say, for each Check, what it looks for, how to fix a failure, and how to mark a
// false alarm.
var explanations = map[string]string{
	"conventional-commits": `conventional-commits (commit-msg) looks for a commit message that is a
Conventional Commit, as commitlint's config-conventional reads it: a "type(scope)!: description"
subject with one of the types and scopes in .hathi.yml, a description not starting in uppercase nor
ending in ".", a blank line before the body, lines of up to max-length characters, and a breaking
change footer as "BREAKING CHANGE: description". Lines starting with # don't count, and a message
Git writes itself, such as a merge's or a revert's, passes. With pr-title on, hathi check ci checks
a pull request's title the same way.

To fix it, rewrite the message: git commit --amend for the last commit, git rebase -i for an older
one, or edit the pull request's title.

To change what passes, set types, scopes or max-length under conventional-commits in .hathi.yml;
conventional-commits: false turns it off.
`,
	"no-ai-coauthor": `no-ai-coauthor (commit-msg) looks for a line of the commit message that
makes an AI its co-author, a Co-authored-by trailer naming Claude, Copilot, Cursor, Codex, Gemini,
Devin, Aider or Amp, or credits one with it, such as "Generated with Claude Code". A person's
Co-authored-by passes, even one named Claude with a surname.

To fix it, remove the line: git commit --amend for the last commit, git rebase -i for an older one.
Claude Code adds its own unless its settings turn attribution off.

no-ai-coauthor: false in .hathi.yml turns it off.
`,
	"no-secrets-in-commits": `no-secrets-in-commits (pre-commit) looks for a Secret on a line the
changes add (a private key with its body; a key or token of AWS, GitHub, GitLab, Slack, Stripe
live, Anthropic, OpenAI, Google, npm, PyPI, Docker Hub, Hugging Face, SendGrid, DigitalOcean,
Shopify, Vault, age or a Telegram bot; a Slack webhook URL, a JWT, a password in a URL) and for an
Env file they add or change, such as .env, .env.local or .envrc; a template of one (.example,
.sample, .template, .dist) passes, and so does a placeholder such as AWS's documented example key,
one of x's, or a ghp_ token whose checksum fails. It names where, never the Secret itself.

To fix it, remove the Secret from the change and keep it in the environment or a secret store;
one already pushed is out, so revoke it too. Keep an Env file out of Git, or commit a template.

To mark a false alarm, put hathi:allow-secret on the line, or on an Env file's first line, whose
lines are still looked through; baloo:allow-secret and gitleaks:allow pass too. A project that
needs deeper scanning runs gitleaks beside hathi.
`,
	"no-conflict-markers": `no-conflict-markers (pre-commit) looks for a conflict marker the changes
add, as git diff --check finds them, in a file where one of them opens or closes a conflict (<<<<<<<
or >>>>>>>): a Markdown heading underlined with ======= alone passes.

To fix it, resolve each conflict and stage the file again.

A file that holds markers on purpose passes with a longer conflict-marker-size in .gitattributes,
such as "testdata/* conflict-marker-size=32".
`,
	"no-large-files": `no-large-files (pre-commit) looks for a file the changes add over max-size
KB, 1024 by default; a file already committed passes, changed or renamed too, and so does a
submodule.

To fix it, keep the file out of Git (git rm --cached <file>, then .gitignore it), or keep it in Git
LFS or a release asset.

To let it in, raise max-size under no-large-files in .hathi.yml.
`,
	"linear-history": `linear-history (pre-push) looks for a merge commit the push sends, and over
a range for a merge commit in it; the merges the remote's default branch already has pass.

To fix it, rebase instead of merging (git rebase <branch>), then push the rebased branch with
git push --force-with-lease. git config pull.rebase true makes git pull rebase.

linear-history: false in .hathi.yml turns it off.
`,
}

// explain is `hathi explain <check>`.
func explain(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || explanations[args[0]] == "" {
		fmt.Fprintf(stderr, "hathi: usage: hathi explain <%s>\n", strings.Join(settings.Checks, "|"))
		return 2
	}
	fmt.Fprint(stdout, explanations[args[0]])
	return 0
}
