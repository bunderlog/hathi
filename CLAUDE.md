# hathi

Commit and repo Checks for Git hooks and CI, as a Go binary. What it is: [README.md](README.md).
Check a change with `mise run check`.

## Where things are decided

- Before designing or planning: the ADRs in `.about/adr/`. Follow them; to change one, edit it in
  place (baloo `adr` skill), never work around it.
- What `hathi` must do: `.about/prd/hathi.md`.
- Naming anything in code, UI or docs: `.about/glossary.md`. One word per concept, the glossary's.
- Open questions: `.about/inbox.md`. Answer from there, don't settle them in code.

## Where the Checks live

Here (ADR distribution): 1.0 ported them from the `baloo` plugin in
[bunderlog/claude-plugins](https://github.com/bunderlog/claude-plugins) at v0.34.0, and `baloo`
drops each one `hathi` has. A Check changes here, not in `baloo` first; a `// baloo:` comment names
the `baloo` test a test was ported from.
