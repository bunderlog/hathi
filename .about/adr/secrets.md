# hathi finds a few kinds of Secret, and gitleaks beside it finds the rest

Date: 2026-11-06

`no-secrets-in-commits` finds Env files by their name, and Secrets of a kind with a published
format whose prefix hardly ever matches anything else (`AKIA`, `ghp_`, `sk-ant-`). A new kind is
added only on those terms; entropy and generic rules, such as gitleaks' `generic-api-key`, are
not. A project that needs deeper scanning runs gitleaks beside `hathi`, in its own Git hooks and
CI: `hathi` neither runs gitleaks nor links it, so a commit gets the same Verdict on every machine
and the binary keeps its few dependencies.

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

## Consequences

- The README shows how to run gitleaks beside `hathi`.
- Revisit when a Secret of a kind with its own prefix gets committed past `hathi`: that kind is
  added, not the rule changed.
