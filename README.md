# hathi

Commit and repo checks for Git hooks and CI: Conventional Commits, no secrets, no AI co-authors,
no conflict markers, no large files, linear history. The same verdict on your machine, in a Git
hook and in CI, whether or not the commit came from Claude Code.

Hathi keeps the Water Truce: at the watering hole every animal obeys the same law, and the
elephant never forgets.

Part of [bunderlog](https://github.com/bunderlog).

## Status

1.0 is released. The Checks are ported from the `baloo` plugin
([bunderlog/claude-plugins](https://github.com/bunderlog/claude-plugins)) at v0.34.0, into a Go
binary that needs no Claude and no runtime.

## Use

Pin it in `mise.toml`, to an exact version. For a release's first 24 hours mise holds it back from
a pin such as `1.0`, which gets the release before it instead; after a first release there is none,
and `mise install` fails.

```toml
[tools]
"github:bunderlog/hathi" = "1.0.1"
```

Commit the Git hooks, and turn them on once per clone. `.githooks/commit-msg` is below;
`pre-commit` and `pre-push` are alike, and `hathi hooks print <hook>` prints each:

```sh
#!/bin/sh
exec hathi check commit-msg "$@"
```

```sh
git config core.hooksPath .githooks
```

In GitHub Actions, `hathi check ci` finds the new commits of the push or PR by itself, given the
full history; elsewhere, `hathi check --range "$BASE..$HEAD"`:

```yaml
- uses: actions/checkout@v7
  with:
    fetch-depth: 0
- uses: jdx/mise-action@v5
- run: hathi check ci
```

Every Check is on without settings; `hathi config --default > .hathi.yml` writes them all out to
change. `hathi explain <check>` says how to fix a failure, `hathi doctor` what is wrong with how
the repo runs `hathi`, and `hathi help` lists the rest.

### Deeper secret scanning

`no-secrets-in-commits` finds Env files and the kinds of key with a prefix of their own, such as
AWS's or GitHub's. For more, run [gitleaks](https://github.com/gitleaks/gitleaks) beside `hathi`,
and keep both on: only `hathi` flags an Env file. `gitleaks:allow` marks a false alarm for both.

```toml
[tools]
"github:bunderlog/hathi" = "1.0.1"
gitleaks = "8.30.1"
```

```sh
#!/bin/sh
# .githooks/pre-commit
hathi check pre-commit "$@" || exit
exec gitleaks git --pre-commit --staged --redact --verbose
```

```yaml
- run: gitleaks git --redact
```

## Development

`mise run check` lints (golangci-lint with gofmt and goimports, actionlint, zizmor) and tests; a
tag `v<version>` makes the Release. `baloo`'s Git hooks also run gitleaks and the lint before a
commit, and the tests before a push (`.claude/baloo.yml`).

## GitHub repo

- Description: Commit and repo checks for Git hooks and CI: Conventional Commits, no secrets, no AI
  co-authors, no conflict markers, no large files, linear history. One law at the watering hole,
  Claude or not.
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
- [.about/adr/secrets.md](.about/adr/secrets.md) — `hathi` finds a few kinds of Secret, and
  gitleaks beside it finds the rest.
- [.about/glossary.md](.about/glossary.md) — the terms.
