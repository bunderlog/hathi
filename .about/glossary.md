# hathi glossary

`hathi`: commit and repo Checks for Git hooks and CI, as a Go binary. Terms taken from the `baloo`
plugin and not defined here keep the meaning its glossary gives them
([bunderlog/claude-plugins](https://github.com/bunderlog/claude-plugins), `.about/glossary.md`).

**Check**:
One deterministic rule over a commit or the repo that passes or fails, such as
`conventional-commits` or `linear-history`.

**Git hook**:
A script Git runs at a step of its own (`commit-msg`, `pre-commit`, `pre-push`), as in `baloo`.
`hathi` has no Hooks of Claude Code, so "hook" alone means a Git hook here, unlike in `baloo`.

**Settings**:
What a repo sets for `hathi`: `.hathi.yml` at its root, or over a Range each commit's own for the
commit-msg and pre-commit Checks and the head's for the pre-push ones (ADR config), turning each
Check on or off and giving its options.
_Avoid_: config, which in `baloo` is `.claude/baloo.yml`
_In code_: the flag `--config` and the command `hathi config` keep the names CLIs use

**Settings problem**:
Anything in the text of a Settings file that `hathi` can't take: text that isn't YAML or isn't a
mapping, a key that isn't a setting, or a value that doesn't decode. `hathi check` stops on one;
`hathi list` and `hathi doctor` report it beside the settings that still apply. A file `--config`
names that doesn't exist is a usage error, not a Settings problem.
_Avoid_: settings error
_In code_: `settings.Problem`

**Input**:
What one Check reads in one run: a commit's message or the PR title, a commit's changes or the
staged ones, or the pushed refs. Over a Range, a pre-push Check's Input is the whole Range, not
each commit.
_In code_: `checks.Input`

**Finding**:
One thing a Check fails on in one Input, such as a commit's changes: where, when the Check can
name a place, and why. An Input that passes has none.
_In code_: `checks.Finding`

**Verdict**:
What `hathi check` reports for one Check on one Input: a failing one per Finding, or a single
passing one where there is none. A Check that finds two Secrets gives two failing Verdicts; one
the Settings turn off gives none; one that can't check, as where Git fails, gives an error
instead, and `hathi check` exits `2`.
_In code_: `cli.Verdict`

**Range**:
The commits `hathi check ci` or `--range` checks one by one, each with the Verdicts its Git hooks
would have given it: those from the merge base of a base and a head to the head. `<base>..<head>`
only names one; for a push to a branch other than the default one, `ci` leaves out what the
default branch already has.
_In code_: `checks.Range`

**Secret**:
A value that grants access: an API key, token, password or private key. A Placeholder, a published
example or a key meant to be public (a Stripe publishable key) isn't one; `no-secrets-in-commits`
finds only a Secret of a Kind. An Env file counts as holding Secrets unless an Allow mark is on its
first line.
_Avoid_: credential, key (alone)

**Env file**:
A file of environment settings a program or shell loads: `.env`, `.env.local`, `prod.env`,
direnv's `.envrc`… A template of one (`.env.example`, `.sample`, `.template`, `.dist`) isn't one.
_In code_: `secrets.IsEnvFile`

**Kind**:
A form of Secret with a published format that hardly anything else shares (ADR secrets): a prefix,
such as an AWS access key's `AKIA` or a GitHub token's `ghp_`, or a fixed shape, such as a private
key's PEM body or a password between `://user:` and `@`. Its name is the reason of a Finding on it.
Not an Env file, which `no-secrets-in-commits` finds by its name, not by what it holds.
_In code_: `secrets.kinds`, `secrets.checksummed`, and a private key apart in `secrets.scan`

**Placeholder**:
A match of a Kind that is no real Secret, and passes: it holds a word such as `example` or
`xxxxxx`, or it fails its Kind's checksum. Not an Allow mark, which a person puts on a real-looking
line to pass it.
_Avoid_: fake
_In code_: `isPlaceholder` for the words, each checksummed Kind's `holds` for the checksum

**Allow mark**:
A comment that passes a line for `no-secrets-in-commits`, or an Env file when on its first line:
`hathi:allow-secret`, and `baloo:allow-secret` and `gitleaks:allow` too.
_Avoid_: marker, the word of a conflict marker
_In code_: `secrets.Allow`

**Release**:
A tagged version of `hathi` on GitHub, carrying an archive with the binary per platform and their
`SHA256SUMS`. How `hathi` reaches a machine or CI; `go install` builds the same version from its
source.
_Avoid_: publish, package
