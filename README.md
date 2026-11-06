# hathi

Commit and repo checks for Git hooks and CI: Conventional Commits, no secrets, no AI co-authors,
no conflict markers, no large files, linear history. The same verdict on your machine, in a Git
hook and in CI, whether or not the commit came from Claude Code.

Hathi keeps the Water Truce: at the watering hole every animal obeys the same law, and the
elephant never forgets.

Part of [bunderlog](https://github.com/bunderlog).

## Status

Pre-start, 2026-11-03. Nothing is built yet. The Checks exist today inside the `baloo` plugin
([bunderlog/claude-plugins](https://github.com/bunderlog/claude-plugins)); `hathi` takes them out
into a Go binary that needs no Claude and no runtime.

## Planned use

Pin it in `mise.toml`, commit the hooks, and turn them on once per clone:

```sh
# .githooks/commit-msg
#!/bin/sh
exec hathi check commit-msg "$@"
```

```sh
git config core.hooksPath .githooks
```

In GitHub Actions, `hathi check ci` finds the new commits of the push or PR by itself; elsewhere,
`hathi check --range "$BASE..$HEAD"`.

## GitHub repo

- Description: Commit and repo checks for Git hooks and CI: Conventional Commits, no secrets, no AI
  co-authors, no large files, linear history. One law at the watering hole, Claude or not.
- Shorter, without the book: Commit and repo checks for Git hooks and CI, with or without Claude
  Code. Part of bunderlog.
- Topics: `git-hooks`, `conventional-commits`, `commit-lint`, `secrets-detection`, `ci`,
  `claude-code`.

## Where things are

- [.about/prd/hathi.md](.about/prd/hathi.md) — what 1.0 must do.
- [.about/adr/distribution.md](.about/adr/distribution.md) — a Go binary through GitHub Releases
  only, and why.
- [.about/adr/hooks.md](.about/adr/hooks.md) — `hathi` runs inside Git hooks and never installs
  them.
- [.about/adr/config.md](.about/adr/config.md) — the settings are `.hathi.yml` at the repo root.
- [.about/glossary.md](.about/glossary.md) — the terms.
- [.about/inbox.md](.about/inbox.md) — open questions.
