# Inbox

## How deep hathi looks for Secrets, and where gitleaks takes over

2026-11-03 · design session · kept 2026-11-06

`no-secrets-in-commits` in `baloo` catches Env files and obvious key patterns. Dedicated scanners
such as gitleaks keep hundreds of maintained rules that `hathi` will not match. The leaning: `hathi`
keeps the simple Check, with no dependencies and the same Verdict everywhere, and a project that
needs deep scanning adds gitleaks to CI beside `hathi`, one more line in `mise.toml`. 1.0 finds what
`baloo` v0.34.0 finds (PRD, Non-goals). Open after 1.0: which patterns `hathi` covers, whether its
README recommends gitleaks, and whether `hathi` should run gitleaks when it is installed. Settled in
an ADR of its own.
