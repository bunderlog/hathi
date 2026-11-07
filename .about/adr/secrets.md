# hathi finds a few kinds of Secret, and gitleaks beside it finds the rest

Date: 2026-11-07

`no-secrets-in-commits` finds Env files by their name, and Secrets of a kind with a published
format whose prefix hardly ever matches anything else (`AKIA`, `ghp_`, `sk-ant-`). A new kind is
added only on those terms; entropy and generic rules, such as gitleaks' `generic-api-key`, are
not. A project that needs deeper scanning runs gitleaks beside `hathi`, in its own Git hooks and
CI: `hathi` neither runs gitleaks nor links it, so a commit gets the same Verdict on every machine
and the binary keeps its few dependencies.

A match is a Secret only if it isn't a placeholder: one holding `example`, `sample`, `xxxxxx`,
`aaaaaa`, `abcde`, `00000` or `*****`, in any case, passes, as trufflehog's filter does, so AWS's
documented example key (`AKIA…EXAMPLE`) and a `ghp_xxxx…` in docs or tests pass. Where a kind's
format has a marker of its own, the pattern asks for it: OpenAI's `T3BlbkFJ`, AWS's base32
alphabet, and a private key's body after its `BEGIN` line, not the line alone. A private key in
base64, as a kubeconfig's `client-key-data` holds one, counts once decoded; it isn't held to the
placeholders, since base64 holds runs such as `AAAAAA`.

A classic GitHub token (`ghp_` and 36 characters) ends in a checksum: the CRC32 of its 30 random
characters in base62 (`0-9A-Za-z`), padded to 6 with `0`, which a real, revoked token confirmed on
2026-11-07, as neither gitleaks nor trufflehog checks. One whose checksum fails passes, so a fake
`ghp_` in tests and docs does. `gho_`, `ghu_`, `ghs_` and `ghr_` likely share it but no token of
theirs has confirmed it, and a fine-grained `github_pat_` has none; they are found without it,
since a wrong guess would pass real tokens. An npm token (`npm_` and 36 characters) has the same
checksum, which a real, revoked one confirmed on 2026-11-07.

A Cloudflare token (`cfk_`, `cfut_`, `cfat_`, then 40 characters) ends in the CRC32 of those 40
in 8 lowercase hex digits. Cloudflare documents the three formats and a CRC32 suffix, and a real
`cfat_` token confirmed the encoding on 2026-11-07, so all three are checked and one whose checksum
fails passes.

A project that adds gitleaks keeps `no-secrets-in-commits` on: gitleaks doesn't flag an Env file by
its name. A Secret both find is reported twice. `gitleaks:allow` marks a false alarm for `hathi`
too, so one mark serves both; gitleaks takes only its own.

## Considered options

- `hathi` running gitleaks when it is on the PATH, its findings turned into Findings — one output
  and one exit code, but the Verdict would depend on what the machine has installed, and `hathi`
  would parse another tool's output across its versions.
- gitleaks as a Go library, or its rules copied — its depth in one binary, but heavy dependencies
  and some 200 rules to keep in step.
- Only the kinds `baloo` v0.34.0 finds — a new provider's key would pass wherever gitleaks doesn't
  run.
- An entropy threshold on a prefixed match, as gitleaks sets even on `ghp_`, to drop placeholders —
  the list of placeholders drops them too, and says why a match passed.
- Checking a token with its issuer's API, as trufflehog does — it needs the network, so the Verdict
  would depend on where `hathi` runs.

## Consequences

- The README shows how to run gitleaks beside `hathi`.
- Revisit when a Secret of a kind with its own prefix gets committed past `hathi`: that kind is
  added, not the rule changed.
