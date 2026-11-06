# hathi

Commit and repo Checks for Git hooks and CI, as a Go binary. What it is: [README.md](README.md).
Check a change with `mise run check` (golangci-lint, `go test`).

## Where things are decided

- Before designing or planning: the ADRs in `.about/adr/`. Follow them; to change one, edit it in
  place (baloo `adr` skill), never work around it.
- What 1.0 must do: `.about/prd/hathi.md`.
- Naming anything in code, UI or docs: `.about/glossary.md`. One word per concept, the glossary's.
- Open questions: `.about/inbox.md`. Answer from there, don't settle them in code.

## Where the Checks come from

The Checks are ported from the `baloo` plugin in
[bunderlog/claude-plugins](https://github.com/bunderlog/claude-plugins), at the Release the PRD
names: its code in `plugins/baloo/scripts/baloo/checks/`, its ADRs `checks`, `git-hooks` and
`binary` in `.about/adr/`, and its test cases in `tests/`, which `hathi` must agree with. Read them
there, or in a local clone if one is at hand, before porting a Check.
