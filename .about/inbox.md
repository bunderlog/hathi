## A Check that a commit's type matches the files it changes
2026-11-27 · chainofsale session
In chainofsale, a `fix(store):` commit failed `conventional-commits` (123 characters, over 100), its
changes stayed staged, and the next commit, `docs: note which Publishers at risk…`, took them along
with its own `.about/inbox.md`: a `docs` commit holding Go code and a test, which every Check
passed, as none compares a commit's type with what it changes. A Check could fail a `docs` commit
that changes anything but documentation (`*.md`, `.about/`, `docs/`), and perhaps a `test` commit
that changes code outside tests. Its Input is a commit's message and its changes together, which
no Check reads yet: at `commit-msg` the message and the staged files are both at hand, and over a
Range each commit's own. Open: which types it rules on (`docs` alone, or `test`, `ci`, `build`
too); what counts as documentation, as a setting with a default; and whether it belongs in
`hathi` or in a review. Settled by deciding those, then a PRD criterion and the Check.
