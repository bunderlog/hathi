# hathi runs inside Git hooks and never installs them

Date: 2026-11-06

`hathi check <hook>` is what a Git hook calls; how the hook gets there is the project's choice.
The recommended way is a folder of short hook scripts committed in the repo, such as `.githooks/`,
each calling `hathi`, turned on once per clone with `git config core.hooksPath .githooks`. husky,
lefthook or a plain `.git/hooks/` script work the same way: each just runs `hathi`.

`hathi` writes no hook, changes no Git config and touches nothing outside its own output. A
project's hooks often run more than the Checks (its own lint or tests), and a tool that wrote
hooks would have to merge with whatever is there, as `baloo` does today with husky.

## Considered options

- `hathi hooks install`, writing into `.git/hooks` — untracked, so each clone must run it anyway,
  and it must merge with hooks it did not write.
- `hathi` setting `core.hooksPath` itself — changes a repo's Git config as a side effect of a tool
  that otherwise only reads.

## Consequences

- The README shows the hook scripts to commit; a project template can carry them.
- A clone that never turns its hooks on runs no Checks locally; CI, which runs `hathi` over the
  pushed range, catches what it commits.
