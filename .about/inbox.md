# Inbox

## Check a GitHub token's checksum offline

2026-11-07 · secrets research

GitHub says the last 6 characters of a `ghp_`-style token are a CRC32 checksum, so `hathi` could
tell a real token from a fake one without the network; neither gitleaks nor trufflehog does. What
the CRC covers (the 30 random characters, or the prefix too) and how it is encoded is not
documented. Settled by checking the algorithm against a real, revoked token: if it holds, a
`ghp_` match whose checksum fails passes as a placeholder (ADR secrets).
