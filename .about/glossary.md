# hathi glossary

`hathi`: commit and repo Checks for Git hooks and CI, as a Go binary. Terms taken from the `baloo`
plugin keep the meaning its glossary gives them
([bunderlog/claude-plugins](https://github.com/bunderlog/claude-plugins), `.about/glossary.md`).

**Check**:
One deterministic rule over a commit or the repo that passes or fails, such as
`conventional-commits` or `linear-history`.

**Git hook**:
A script Git runs at a step of its own (`commit-msg`, `pre-commit`, `pre-push`), as in `baloo`.
`hathi` has no Hooks of Claude Code, so "hook" alone means a Git hook here, unlike in `baloo`.

**Settings**:
What a repo sets for `hathi`: `.hathi.yml` at its root, or over a Range each commit's own (ADR
config), turning each Check on or off and giving its options.
_Avoid_: config, which in `baloo` is `.claude/baloo.yml`
_In code_: the flag `--config` and the command `hathi config` keep the names CLIs use

**Settings problem**:
A key of the Settings that isn't a setting, or a value that doesn't decode. `hathi check` stops on
one; `hathi list` and `hathi doctor` report it beside the settings that still apply.
_Avoid_: settings error
_In code_: `settings.Problem`

**Finding**:
One thing a Check fails on in one input, such as a commit's changes: where, when the Check can
name a place, and why. An input that passes has none.
_In code_: `checks.Finding`

**Verdict**:
What `hathi check` reports for one Check on one input: a failing one per Finding, or a single
passing one where there is none. A Check that finds two Secrets gives two failing Verdicts; one
the Settings turn off gives none.
_In code_: `cli.Verdict`

**Range**:
The commits `hathi check ci` or `--range` checks one by one, each with the Verdicts its Git hooks
would have given it: those from the merge base of a base and a head to the head. `<base>..<head>`
only names one; for a push to a branch other than the default one, `ci` leaves out what the
default branch already has.

**Release**:
A tagged version of `hathi` on GitHub, carrying an archive with the binary per platform and their
`SHA256SUMS`. How `hathi` reaches a machine or CI; `go install` builds the same version from its
source.
_Avoid_: publish, package
