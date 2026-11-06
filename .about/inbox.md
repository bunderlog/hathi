# Inbox

## How deep hathi looks for Secrets, and where gitleaks takes over

2026-11-03 · design session · kept 2026-11-03

`no-secrets-in-commits` in `baloo` catches env files and obvious key patterns. Dedicated scanners
such as gitleaks keep hundreds of maintained rules that `hathi` will not match. The leaning: `hathi`
keeps the simple Check, with no dependencies and the same Verdict everywhere, and a project that
needs deep scanning adds gitleaks to CI beside `hathi`, one more line in `mise.toml`. Open: which
patterns `hathi` covers, whether its README recommends gitleaks, and whether `hathi` should run
gitleaks when it is installed. Settled in an ADR of its own.

## Conventional Commits in hathi, and where commitlint takes over

2026-11-03 · design session · kept 2026-11-03

Unlike Secrets, Conventional Commits is a small, closed spec, so `hathi`'s own Check can be
complete: `baloo`'s already follows commitlint's `config-conventional`, with types and the longest
line configurable, and parity with it is testable case by case. commitlint needs Node and a config
file; cocogitto and convco are binaries but bring changelogs and versioning `hathi` doesn't need.
The leaning: `hathi` keeps its own Check, matching `config-conventional` defaults plus types, scopes
and line length; a project that needs commitlint's plugins or shared configs runs commitlint and
turns `hathi`'s Check off. One project already runs commitlint with husky. Open: whether `hathi`
reads a commitlint config when it finds one, or ignores it and leaves the choice to the project.
Settled in an ADR of its own.
