---
date: 2026-06-19
type: success
status: validated
related_specs:
  - ../specs/019c0a6b28007abc-tf-drift-cli.md
---

# Version Reporting Success

## Context

Users expect `tf-drift -version` to print the released version, such as `v1.0.0`, and also expect the common short `-v` alias to work.

## Intended Outcome

Release binaries should print the exact GoReleaser tag, while source builds should still show useful git-derived version metadata instead of plain `dev`.

## What Worked

The CLI keeps version reporting build-time driven. GoReleaser already injects `main.version` from the release tag, and local Makefile builds now inject `git describe --tags --always --dirty`. The `-v` alias shares the same flag target as `-version`, so both paths stay identical.

## Evidence

Focused tests cover the long flag, short flag, injected version, and fallback version behavior. A local `make build` verified that both `./tf-drift -version` and `./tf-drift -v` print the same git-derived version string.

## Reusable Pattern

Use build-time ldflags for release metadata. Avoid runtime git calls in CLI binaries because installed release artifacts may not have a `.git` directory.

## Limits

Dirty source builds include a `-dirty` suffix. Official GoReleaser releases remain clean tag values such as `v1.0.0` because they use the release tag injected by GoReleaser.
