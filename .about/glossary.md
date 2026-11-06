# hathi glossary

`hathi`: commit and repo Checks for Git hooks and CI, as a Go binary. Terms taken from the `baloo`
plugin keep the meaning its glossary gives them
([bunderlog/claude-plugins](https://github.com/bunderlog/claude-plugins), `.about/glossary.md`).

**Check**:
One deterministic rule over a commit or the repo that passes or fails, such as
`conventional-commits` or `linear-history`. The same meaning as in `baloo`.

**Verdict**:
What a Check returns, one per finding: fail with one line naming the Check and the reason, or
pass for an input it finds nothing in. A Check that finds two Secrets gives two Verdicts.
`hathi` and `baloo` must give the same Verdicts for the same input, the reason's wording aside
(PRD, criterion 10).

**Release**:
A tagged version of `hathi` on GitHub, carrying an archive with the binary per platform and their
`SHA256SUMS`. How `hathi` reaches a machine or CI; `go install` builds the same version from its
source.
_Avoid_: publish, package
