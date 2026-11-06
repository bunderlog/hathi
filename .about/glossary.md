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
What a repo sets for `hathi`: `.hathi.yml` at its root, or over a range each commit's own (ADR
config), turning each Check on or off and giving its options.
_Avoid_: config, which in `baloo` is `.claude/baloo.yml`
_In code_: the flag `--config` and the command `hathi config` keep the names CLIs use

**Verdict**:
What a Check returns, one per finding: fail with one line naming the Check and the reason, or
pass for an input it finds nothing in. A Check that finds two Secrets gives two Verdicts.

**Release**:
A tagged version of `hathi` on GitHub, carrying an archive with the binary per platform and their
`SHA256SUMS`. How `hathi` reaches a machine or CI; `go install` builds the same version from its
source.
_Avoid_: publish, package
