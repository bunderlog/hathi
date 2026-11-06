# hathi 1.0

## Problem

The commit and repo Checks run only where Claude Code runs with the `baloo` plugin: the plugin
writes the Git hooks at session start. A commit by a person without the plugin, an edit in GitHub's
web editor, or a commit from another tool passes unchecked. CI can run the Checks, but only by
downloading a `baloo` Release by its commit and running its `run.py` with each Check's input
written out by hand.

## Who it is for

- Andrew's own projects, the `bunderlog` repos among them, in Git hooks and in CI.
- Anyone who wants the Checks without the rest of `baloo`'s method.

## Success

Every project's CI runs the Checks in one step from a version pinned in `mise.toml`, and a commit
that fails a Check fails the same way locally, in Git hooks and in CI.

## Scope

The Checks that are about commits and the repo, as `baloo` v0.34.0 defines them (its
`schema.json`, `git-hooks`):

| Check | Git hook | What fails |
| --- | --- | --- |
| `conventional-commits` | commit-msg | a message that is not a Conventional Commit |
| `no-ai-coauthor` | commit-msg | an AI as co-author, or a line crediting one; a person's `Co-Authored-By` passes |
| `no-secrets-in-commits` | pre-commit | staged changes that add a Secret or an Env file; `hathi:allow-secret` marks a false alarm |
| `no-conflict-markers` | pre-commit | staged changes that add a conflict marker |
| `no-large-files` | pre-commit | a new file over `max-size` |
| `linear-history` | pre-push | a push that sends a merge commit |

Each Check's rules are `baloo`'s code in `plugins/baloo/scripts/baloo/checks/` at v0.34.0, the
Release `hathi` ports from, its docstrings included: among them, a message's `#` lines and what
follows a `git commit -v` scissors line are ignored, a message Git writes itself (`Merge …`,
`Revert "…"`, `fixup! …`) passes `conventional-commits` but not `no-ai-coauthor`, an Env file's
template (`.example`, `.sample`, `.template`, `.dist`) passes, the marker goes on the line or on an
Env file's first line, and a "new file" is one the changes add, not one they grow. `explain` and the
docs name `hathi:allow-secret`; `baloo:allow-secret` passes too, so lines already marked still pass.

Settings in `.hathi.yml` are `baloo`'s keys under `git-hooks`, at the top level, plus `scopes` and
`pr-title` under `conventional-commits`: each Check's name with `true`, `false` or its settings. A
Check without its key is on with its defaults, unlike in `baloo`. The settings and their defaults:

- `conventional-commits`: `types`, config-conventional's 11 (`build`, `chore`, `ci`, `docs`,
  `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test`) or `any`; `scopes`, `any` or the
  scopes a subject may have, matched as commitlint's `scope-enum` matches them: `feat(api,ui)`
  split at `,`, `/` or `\`, each part in the list, case counting, and a subject without a scope
  passing either way; `max-length`, the longest line, 100, `0` for no limit; `pr-title`, `false`.
- `no-large-files`: `max-size`, 1024, in KB of 1024 bytes; a file fails when it is larger.

`types: any` and `scopes: any` are the string, not a list, a `scopes` list holds at least one,
and `max-size` is at least 1.
`no-stale-adr-date` copied from a `baloo.yml` is an unknown key, whose error says it stays in
`baloo`. `hathi config --default` prints the `$schema` line first (ADR config).

## Command line

One binary with subcommands (ADR distribution).

| Command | Called by | Does |
| --- | --- | --- |
| `hathi check commit-msg <file>` | Git hook | checks the message in `<file>`, or on stdin for `-`: `conventional-commits`, `no-ai-coauthor` |
| `hathi check pre-commit` | Git hook | checks the staged changes: `no-secrets-in-commits`, `no-conflict-markers`, `no-large-files` |
| `hathi check pre-push [<remote> <url>]` | Git hook | reads the pushed refs on stdin as Git gives them: `linear-history`; the arguments Git passes are optional and ignored, as `baloo` compares with the remote-tracking branches |
| `hathi check ci` | CI | finds the new commits in GitHub Actions and checks each one's message and changes, and merge commits |
| `hathi check --range <base>..<head>` | other CIs, a person | the same over a range given by hand |
| `hathi list [--format text\|json]` | a person | every Check: its hook, on or off in this project, its settings; a settings problem is listed too and exits `2` (ADR config) |
| `hathi explain <check>` | a person, Claude | what the Check looks for, how to fix a failure, how to mark a false alarm |
| `hathi doctor [--format text\|json]` | a person, Claude | settings problems; whether each hook with a Check on (all three by default) is in the folder Git runs hooks from (`git rev-parse --git-path hooks`) and has a line with `hathi check <hook>` in it (`mise exec -- hathi check …` too), there, in `.husky/<hook>`, or under `<hook>` in `lefthook.yml` or `.lefthook.yml`; whether the running version matches the `github:bunderlog/hathi` pin in `mise.toml` or else `.mise.toml` at the root, as a string or `{ version = … }`, a pin matching by whole parts with or without a leading `v` (`1` any `1.x.y`, `1.0` any `1.0.x`; with no pin, or `latest`, it says so and that is no problem); changes nothing, exits `1` when it finds a problem, `2` for a settings problem (ADR config) |
| `hathi hooks print <hook>` | a template, a person | prints the recommended hook script for `commit-msg`, `pre-commit` or `pre-push` to stdout, `#!/bin/sh` and `exec hathi check <hook> "$@"`; writes no file (ADR hooks) |
| `hathi config --default` | a person | prints the default settings with every key and a comment; writes no file |
| `hathi version`, `hathi --version` | a person, mise | prints its own version, such as `1.0.0`, or `devel` for a build without one, which `doctor` reports as no problem |
| `hathi help [<command>]`, `-h`, `--help` | a person | prints the usage of `hathi` or of one command |

`list --format json` prints a JSON object per line, one per Check, with `check`, `hook`, `on` and
`settings`, and one per settings problem, with `problem` and `line`; `doctor --format json` one per
thing it checked, with `topic`, `problem` (`true` or `false`) and `message`.

Every command that reads the settings takes `--config <path>` (ADR config), the path from the
current folder; a path with no file there exits `2`. Outside a Git repo every command but `version`,
`help`, `explain`, `hooks print` and `config --default` exits `2`. Only `text` is ever in color, and
only when the stream it goes to is a terminal and `NO_COLOR` is not set.

Flags for `check`:

- `--only <check>`, `--skip <check>` — run or leave out Checks without editing the settings; each
  may be repeated. `--only` runs a Check even where the settings turn it off;
  `--skip conventional-commits` skips the PR title too. Both at once, an unknown Check, or a Check
  of another hook (`check commit-msg --only no-large-files`) is a usage error; `ci` and `--range`
  take every Check.
- `--format text|json|github` — text for people, JSON for Claude and scripts, GitHub Actions
  annotations so a failure shows on its line in the PR. The default is `github` when
  `GITHUB_ACTIONS` is `true`, else `text`.

What `check` prints, a Verdict per finding (glossary); a passing one's `reason` is empty. A Check
off in the settings or skipped gives none. Over a range each Check gives its Verdicts per commit; a
push without a merge commit, like a range without one, gets a single passing `linear-history`
Verdict. A failing Verdict splits `baloo`'s finding: `file` and `line` where it names them, the rest
as `reason` (`a conflict marker` where it names only the place); for `no-ai-coauthor`, `line` is the
line of the message and `reason` its text; for `linear-history`, `commit` is the merge commit and
`reason` the ref pushed, or over a range the 12-character id of its head. `commit` is a 12-character
id, as in `baloo`.

- `text`, on stderr, as a Git hook shows it: nothing when every Check passes, so a Git hook stays
  quiet; for each failure one line,
  `<commit> <check>: <file>:<line>: <reason>; run: hathi explain <check>`, with `PR title` for
  `<commit>` on the PR's title, and leaving out `<commit> `, `<file>:` and `<line>: ` where the
  Verdict has none. What `baloo` prints as advice before its findings is `explain`'s.
- `json`, on stdout: a JSON object per line, one per Verdict, passing ones too, with `check`,
  `verdict` (`pass` or `fail`) and `reason`, and `commit`, `file` and `line` where they apply.
- `github`, on stdout: one `::error file=…,line=…,title=<check>::<reason>` per failure, escaped as
  GitHub's workflow commands require. A failure with no file, such as a message's or the PR
  title's, leaves out `file` and `line` and has `title=<check> <commit>`, or
  `title=<check> PR title` for the PR's, whose JSON has no `commit`.

`hathi check` with neither a hook nor `--range` is a usage error, and so is `check ci` where
`GITHUB_ACTIONS` is not `true`, which names `--range` instead. An error that exits `2` goes to
stderr as one line in every format, and in `github` also as an `::error title=hathi::` annotation.

Over a range, `ci` and `--range` take every commit `git rev-list <base>..<head>` gives, a side
branch's too, and give each the Verdicts its Git hooks would have given it: the commit-msg Checks on
its message and the pre-commit Checks on its changes against its parent, as if staged there, with
that commit's `.hathi.yml`, unless `--config` names a file, and its `.gitattributes` (Git 2.40 or
later, for `--attr-source`; with an older Git, `ci` and `--range` exit `2` naming the version,
rather than check with other attributes than the commit's). A merge commit gets the commit-msg
Checks and a failing `linear-history` Verdict, but no pre-commit Checks, as its parents brought its
changes; a range without one gets a single passing `linear-history` Verdict. So a Secret one commit
adds and the next removes fails, as it did locally; `baloo`'s CI recipe, which checks the changes
together, would miss it. They read the commits without touching HEAD, the index or the working tree,
an Env file's first line from the commit. `--range` starts from the merge base of `<base>` and
`<head>` and leaves nothing out, unlike `ci` on a `push`; `<head>` need not be checked out.

`check ci` reads the range from the GitHub Actions event. On `pull_request` it is from the merge
base of the event's `base.sha` and `head.sha` to `head.sha`, so `actions/checkout`'s default
checkout of the PR's merge commit works, given its full history. On `push` it is `before..after`,
from the merge base of `before` and `after` after a force push, and from the merge base of `after`
and `origin/<repository.default_branch>` for a new branch, whose `before` is all zeros, or where the
checkout lacks `before`, as it usually does after a force push; a first push of the default branch
itself checks all its commits. A push of a tag or a deletion checks nothing and exits `0`, and any
other event is a usage error. As in `baloo`'s `linear-history`, a `push` range to any branch but the
default one leaves out what `origin/<repository.default_branch>` has, so a branch rebased onto it
isn't blamed for its commits and merges. A shallow checkout
(`git rev-parse --is-shallow-repository`) exits `2` and names `fetch-depth: 0`, whatever the range;
so does a push to the default branch whose `before` the checkout lacks, as after a force push to it,
which leaves no base to check from. Any other commit of the range that the checkout lacks exits `2`
too, and so does any failure of Git. On `pull_request` it also runs `conventional-commits` on the PR
title when the `pr-title` setting of `head.sha` is on (off by default; turn it on where the repo
allows squash merges, since GitHub takes a squashed commit's message from the PR title). The
projects merge by rebase, so the commits in the range are what lands on `main`; a `push` to `main`
checks what actually landed, a squashed message included.

Exit codes: `0` every Check passed, `1` a Check failed, `2` a usage error or a Settings problem, so
CI tells bad code from a broken run; a Check that can't check, as where Git fails, exits `2` too.
The other Checks and commits still run and print their Verdicts, and `hathi` exits with the worst
code, `2` over `1`, as `baloo`'s Git hooks do.

Left out on purpose: `init` or `install` (`hathi` writes no file and no Git config), `fix`
(rewriting a message is for a person or Claude), `changelog` and `release` (cocogitto and
semantic-release do those).

## Non-goals

- `no-stale-adr-date` — about `.about/`, stays in `baloo`.
- Reading a commitlint config. A project that needs commitlint's plugins or shared configs runs
  commitlint and turns `conventional-commits` off.
- Installing Git hooks. The project wires `hathi` into its own hooks (ADR hooks); `hathi` only runs
  inside them.
- Running gitleaks or matching its rules: 1.0 finds the Secrets `baloo` v0.34.0 finds; deeper
  scanning is gitleaks', run beside `hathi` (ADR secrets).
- Running the project's own commands (`mise run check`); that is the project's hook config.
- Guarding Claude's tool calls; that belongs to a Claude Code plugin, inside the session.
- Testing on Windows: the Windows binary ships, but the Checks are tried on macOS and Linux only.

## Acceptance criteria

1. `hathi check commit-msg <file>` exits `1` with a line per failure, naming the Check, the reason
   and `run: hathi explain <check>`, for a message that fails any enabled commit-msg Check, and `0`
   with no output when it passes. `hathi check commit-msg -` gives the same Verdict for the same
   message on stdin.
   → test: internal/cli/cli_test.go "TestCommitMsg_FailsAMessageWithALinePerFailure";
   internal/cli/cli_test.go "TestCommitMsg_PassesAMessageQuietly"
2. `hathi check pre-commit` exits `1` for staged changes that fail an enabled pre-commit Check and
   `0` when they pass; `hathi check pre-push` does the same for the pushed refs on stdin as Git
   gives them, with or without `<remote> <url>` and whatever they are.
   → test: internal/cli/cli_test.go "TestPreCommit_ChecksTheStagedChanges";
   internal/cli/cli_test.go "TestPrePush_ChecksThePushedRefs"
3. In GitHub Actions, `hathi check ci` with no arguments checks exactly the commits a
   `pull_request` or `push` event brings, and `hathi check --range <base>..<head>` exactly those
   `git rev-list` gives from the merge base of `<base>` and `<head>`: each commit's message and
   changes, and its merge commits; a Secret one commit adds and the next removes fails. A `push` of
   a new branch checks its commits since the default branch, and one after a force push those since
   the merge base of `before` and `after`, or since the default branch where the checkout lacks
   `before`; a branch rebased onto the default branch isn't blamed for the commits it brings. For
   `ci`, a shallow checkout and a push to the default branch whose `before` the checkout lacks exit
   `2`; for both, so does a `<base>` or `<head>` missing from the checkout.
   → test: internal/cli/cli_test.go "TestRange_ChecksEachCommit"; internal/cli/ci_test.go
   "TestCI_ChecksAPullRequestFromTheMergeBase"; internal/cli/ci_test.go
   "TestCI_ChecksANewBranchSinceTheDefaultOne"; internal/cli/ci_test.go "TestCI_ChecksAForcePush";
   internal/cli/ci_test.go "TestCI_ChecksTheDefaultBranch"; internal/cli/ci_test.go
   "TestCI_Exits2WithoutTheHistory"
4. With `pr-title` on, `hathi check ci` on `pull_request` fails for a PR title that is not a
   Conventional Commit; with it off, the title is never checked.
   → test: internal/cli/ci_test.go "TestCI_ChecksThePullRequestTitleWithPRTitleOn"
5. `--format github` turns each failure into a GitHub Actions annotation; `--format json` prints
   one object per Verdict, with `check`, `verdict` and `reason`. Without `--format`, `check` prints
   annotations when `GITHUB_ACTIONS` is `true` and text otherwise.
   → test: internal/cli/cli_test.go "TestCheck_AnnotatesInGitHubActions"; internal/cli/cli_test.go
   "TestRange_PrintsEachVerdictInJSON"
6. A usage error or a Settings problem exits `2`, never `1`: among them `--only` with `--skip`,
   `--only` or `--skip` naming an unknown Check or one of another hook, `hathi check` with neither a
   hook nor `--range`, and `hathi check ci` outside GitHub Actions, whose message names `--range`.
   So does a Check that can't check, as on an unreadable message file or where Git fails.
   → test: internal/cli/cli_test.go "TestRun_Exits2OnAUsageError"; internal/cli/cli_test.go
   "TestCheck_Exits2WhereACheckCantCheck"; internal/cli/cli_test.go
   "TestCheck_OnlyAndSkipPickTheChecks"; internal/cli/cli_test.go
   "TestRange_Exits2WhereGitCantReadACommit"
7. Settings come from `.hathi.yml` at the repo root, or the file `--config` names (ADR config);
   with none, every Check above is on with its defaults, and `hathi config --default` prints exactly
   those defaults. An unknown key or a bad value exits `2` with its line number. Over a range, a
   commit whose `.hathi.yml` turns a Check off is checked with it off, and the commits before it
   with it on.
   → test: internal/settings/settings_test.go "TestParse_DefaultsWithoutSettings";
   internal/settings/settings_test.go "TestParse_TellsEachProblemByItsLine";
   internal/cli/cli_test.go "TestCheck_TakesSettingsFromTheRootOrConfig"; internal/cli/cli_test.go
   "TestConfig_DefaultPrintsTheDefaults"; internal/cli/cli_test.go
   "TestRange_ChecksEachCommitWithItsOwnSettings"
8. `hathi doctor` and `hathi list` change no file and no Git config. `hathi doctor` exits `1` when
   it reports a problem, `2` when one is in the settings, and `0` when it reports none.
   → test: internal/cli/cli_test.go "TestListAndDoctor_ChangeNothing"; internal/cli/cli_test.go
   "TestDoctor_FindsTheGitHooks"
9. Output carries no color codes when the stream it goes to is not a terminal, when `NO_COLOR`
   is set, or in `json` and `github`.
   → test: internal/cli/cli_test.go "TestCheck_PrintsNoColor"
10. For every test in `test_commit.py`, `test_coauthor.py`, `test_secrets.py`,
    `test_conflict.py`, `test_size.py` and `test_history.py` of `tests/checks/` in
    `bunderlog/claude-plugins` at v0.34.0, a Go test names it in a comment,
    `// baloo: <file> <test>`, and gives the same Verdicts as `baloo`: pass or fail, and each
    finding's `file` and `line`, or for `linear-history` its commit, and for `no-ai-coauthor` the
    line it names, not the reason's text. A test of what `hathi` changes on purpose (`mask_secrets`,
    staging `baloo`'s own files) is named in such a comment with why it is left out. One that
    expects an error (`test_fails_outside_a_repo`, `test_fails_on_a_commit_not_here`) expects exit
    `2`, and `test_never_reports_the_secret` that no output holds the Secret;
    `test_githook.py` is left out whole, as it tests `baloo`'s runner, which the Command line here
    replaces.
    → test: each `// baloo:` test in internal/checks/commit/commit_test.go,
    internal/checks/coauthor/coauthor_test.go, internal/checks/secrets/secrets_test.go,
    internal/checks/conflict/conflict_test.go, internal/checks/size/size_test.go,
    internal/checks/history/history_test.go
11. Pushing a tag `v<version>` makes a GitHub Release carrying
    `hathi_<version>_<os>_<arch>.tar.gz` for `darwin` and `linux` on `arm64` and `amd64` and
    `hathi_<version>_windows_amd64.zip`, each holding the one binary, and a `SHA256SUMS` of them.
    On macOS and Linux, `"github:bunderlog/hathi" = "<version>"` under `[tools]` in `mise.toml`
    makes `mise install` install it, and `hathi version` then prints `<version>`; a workflow checks
    it after each Release.
    → test: .github/workflows/release.yml "install", run after each Release
12. `hathi version` and `hathi --version` print the binary's own version, such as `1.0.0`, which
    is the exact pin of criterion 11 when mise installed it; one built by
    `go install …@v<version>` prints `<version>` too.
    → test: internal/cli/cli_test.go "TestVersion_PrintsItWithoutTheV" (a `go install` build's
    version is not tested)
