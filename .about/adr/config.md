# The settings are `.hathi.yml` at the repo root

Date: 2026-11-07

`hathi` reads its settings from one file, `.hathi.yml`, at the root of the repo
(`git rev-parse --show-toplevel`). It looks nowhere else and merges nothing: a monorepo has one file
for the whole repo. Over a range (`hathi check ci`, `--range`), each commit is checked with its own
`.hathi.yml`, as its Git hooks would have checked it, so a commit that loosens a setting doesn't
loosen it for the commits before; a pre-push Check, whose Input is the whole range, takes the
head's, as a push takes the checkout's. `--config <path>` points at another file, for CI or an
unusual layout. Without a file every Check is on with its defaults, as the PRD says.

The file is YAML. Its first line names the JSON Schema for editors,
`# yaml-language-server: $schema=…`, so keys complete and wrong values show as you type, as
`baloo.yml` does today.

A Settings problem (glossary), such as a key that isn't a setting or a value that doesn't decode,
is one line with its line number, exit `2`. The other settings still apply in `hathi doctor` and
`hathi list`, so one typo doesn't hide the rest; `hathi check` stops on it, since a Check run with
settings the project didn't mean would pass or fail for the wrong reason.

`hathi` never writes the file. `hathi config --default` prints a full one to stdout.

## Why the root, not `.claude/`

`hathi` runs without Claude, and `.claude/` is Claude Code's settings folder: a person without
Claude doesn't look there, and a repo without Claude would gain the folder only for `hathi`. Tools
that check a repo keep their settings at its root (`.golangci.yml`, `.gitleaks.toml`,
`.markdownlint.json`, `.lintstagedrc`, `.editorconfig`), so a person opening the repo sees what
checks it. `baloo` keeps `.claude/baloo.yml` beside Claude Code's settings because it is a Claude
Code plugin (its ADR config); `hathi` has no such reason.

## Why YAML

The same format as `baloo.yml`, so the Checks' settings move over almost as they are, and the same
editor support through a schema. Go reads YAML with an ordinary library; the reason `baloo` wrote
its own reader (Python's standard library only) doesn't apply.

## Considered options

- `.claude/hathi.yml` — ties a tool that needs no Claude to Claude's folder.
- `hathi.yml` without the dot — visible, but repo tools' settings are dotfiles by convention.
- TOML (`.hathi.toml`), like `mise.toml` — would do as well, but gives the Checks' settings a
  second format beside `baloo.yml` for nothing in return.
- JSON — no comments, and the default file explains each key in a comment.
- Looking up the tree and merging files, per folder of a monorepo — the Checks are about commits,
  and a commit belongs to the whole repo.
- A section in `mise.toml` — mise has no place for a tool's own settings.

## Consequences

- When `baloo` drops its Checks for `hathi`'s (ADR distribution), their keys under `git-hooks` in
  each project's `.claude/baloo.yml` move to `.hathi.yml`: one migration per project, which `baloo`
  should offer rather than leave to be found.
- The schema is `schema.json` at this repo's root, and `$schema` points at its raw URL on `main`,
  `https://raw.githubusercontent.com/bunderlog/hathi/main/schema.json`. Every project's
  `.hathi.yml` carries that URL, so the file stays where it is.
