# This repo is set up for bunderlog's plugins, and its own Git hooks run the hathi it pins

Date: 2026-11-09

This repo is set up by PRD project-setup in `bunderlog/claude-plugins`: `.claude/settings.json`
adds the `bunderlog` marketplace, turns baloo on and denies Claude reading Env files and keys;
`CLAUDE.md` imports `.hathi.yml`, which lists the commit types in full; and the Git hooks in
`.githooks/` run `hathi check <hook>` first, from the version `mise.toml` pins, then gitleaks and
the lint before a commit and the tests before a push. Each clone turns them on once with `git
config core.hooksPath .githooks`, as the README says.

It moved off what baloo wrote on 2026-11-09: the committed `outputStyle`, the Status line baloo
set in `.claude/settings.local.json`, and baloo's Git hooks in `.git/hooks`, unused since
`.githooks/`. `.claude/baloo.yml` stays, with the Output style, the Status line and every Git
hook Check off, until baloo works without a Config.
