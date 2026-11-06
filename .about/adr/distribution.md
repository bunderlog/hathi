# hathi is a Go binary, shipped through GitHub Releases only

Date: 2026-11-06

`hathi` is written in Go and released on GitHub under `bunderlog/hathi` as an archive per platform
holding its binary (macOS and Linux on arm64 and amd64, Windows on amd64), with a `SHA256SUMS` of
them. GitHub Releases are the only registry:

- **Projects and CI** pin it in `mise.toml`, through mise's backend for GitHub Release assets. CI
  already installs mise (`jdx/mise-action`), so a local run and a CI run use the same version, and
  once installed it works offline.
- **Go users** can `go install` it, which builds from source.
- **A Claude Code plugin** that wants `hathi` treats it like any other tool the project installs;
  how it gets the binary is the plugin's decision, not this one.

It is one binary with subcommands (`hathi check commit-msg`, `hathi check pre-commit`, …), as
`git` or `gh` are, not a binary per Check. The Checks share reading Git (staged changes, a range,
pushed refs on stdin), the settings and the Verdict format, and a project picks Checks in its
settings, not by installing binaries. Each Check is a Go package of its own under
`internal/checks/` behind one interface, so one that grows into a product of its own moves out in
one commit.

npm and PyPI add nothing: everyone who needs `hathi` reaches it through mise, `go install` or the
plugin. The name is free on npm and taken on PyPI (checked 2026-11-03).

## Why Go, against `baloo`'s ADR binary

`baloo` ran a Go binary until 2026-10-14 and moved to Python, standard library only (ADR binary in
`bunderlog/claude-plugins`). Its reasons, and why they don't hold for `hathi`:

- **The binary was opaque where it ran.** `baloo`'s binary was code inside a plugin that users
  never chose to install on its own. `hathi` is a tool people pin by version on purpose, its source
  is public, and `go install` builds it from that source.
- **A build per platform.** A release job cross-compiles every platform from one runner; Go makes
  that a matrix of environment variables, not a toolchain per platform.
- **A Loader and a wait for the download after each update.** `baloo` downloaded the binary at
  session start, inside Claude Code. `hathi` is installed by mise when a project is set up or its
  pin changes, outside any session.
- **No Windows.** That came from the `sh` Loader, not from Go; Go builds Windows like any other
  target.
- **No Checks offline.** mise keeps the installed version; nothing is fetched at run time.

What Go gives `hathi`: no runtime on the machine or in CI (no `python3` in a slim CI image, no
version spread to support), about 2 ms a start in a Git hook that runs on every commit, and one
static file per platform.

## Considered options

- A binary per Check — a Release per Check on every platform, a `mise.toml` line each, versions
  that drift apart, and the shared Git reading copied or turned into a library.
- Python, standard library only, as one zipapp (`hathi.pyz`) per Release — agrees with ADR binary
  and needs no build, but needs `python3` 3.9+ everywhere `hathi` runs, CI images included, which
  is the dependency a standalone tool should not have.
- npm, with prebuilt binaries per platform as optional dependencies (as esbuild and biome do) —
  only useful for people with Node and without mise.
- PyPI under another name (`bunderlog-hathi`) — needs pip or uv on every machine, and `hathi` is
  taken there.
- TypeScript run with Node or Bun — every environment would need that runtime.
- A brew tap — easy later for people outside the family; not needed while mise covers the projects.

## Consequences

- `hathi` needs a release job that builds every platform and attaches the archives and
  `SHA256SUMS`; their names are in the PRD (criterion 11).
- A project's `mise.toml` gains one pinned tool line; a project template can carry it.
- CI runs `hathi` from mise instead of downloading a `baloo` Release and running its `run.py`.
- 1.0 ported the Checks from `baloo`'s Python (`plugins/baloo/scripts/baloo/checks/`) to Go, with
  the same Verdicts, which Go tests mirroring `baloo`'s prove (PRD, criterion 10). Since
  2026-11-06 the Checks live in `hathi` and `baloo` drops each one `hathi` has, so no Check is kept
  in two places and none changes in `baloo` first.
