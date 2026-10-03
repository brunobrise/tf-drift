---
date: 2026-10-03
type: success
status: validated
related_specs:
  - ../specs/01a1039a75fa30ea-v2-drift-engine-evolution.md
evidence_links:
  - ../../internal/drift/runner.go
  - ../../internal/drift/baseline.go
  - ../../internal/drift/remediation.go
  - ../../internal/drift/diff.go
  - ../../internal/drift/lock.go
---

# V2 Drift Engine Evolution Success

## Context

While `tf-drift` classified drift and planned changes effectively, real-world adoption in enterprise repositories faced five barriers:
1. Writing `tfplan` binaries directly into scanned workspaces caused dirty git trees and leaked sensitive plans on abnormal termination.
2. Concurrent deployments holding remote state locks caused unhandled plan failures (`exit code 1`), masking genuine drift.
3. Changed attribute names were displayed without `before` and `after` values, forcing operators back to raw terminal planning.
4. Repositories with pre-existing legacy drift could not adopt `tf-drift` in CI without breaking builds on day one.
5. Operators receiving drift notifications lacked targeted remediation commands.

## Bet

Isolating plan binaries into secure temp directories, classifying state locks into a dedicated `LOCKED` status, preserving `AttributeDiff` before/after values, snapshotting legacy drift via `--baseline`, and exporting targeted remediation recipes via `--export-remediation` solves day-one operational friction without requiring cloud SDK dependencies or complex backend platforms.

## What Worked

* **Zero-Contamination Runner:** `RunPlan` now writes plan binaries to an isolated `os.MkdirTemp` sandbox directory outside the repository tree, ensuring zero `.tfplan` artifacts pollute git working trees.
* **State Lock Classifier:** Detects lock acquisition errors (`isLockErrorOutput`) and maps them to `StateLockError`, reporting `LOCKED` layer status and allowing `-on-locked=skip|fail` with dedicated exit code `4`.
* **Rich Diff Inspector:** `ExtractAttributeDiffs` captures both `Before` and `After` states from plan JSON. Rendered with color-coded diff markers (`-` and `+`) in the TUI inspector pane, Markdown summaries, and human-readable text output.
* **Baseline Snapshot System:** `-baseline <file>` and `-update-baseline` allow teams to snapshot existing known drift into a versioned JSON file. Matching changes are marked `Acknowledged: true`, preventing them from failing CI builds.
* **Remediation Generator:** `GenerateRemediationRecipes` and `-export-remediation <file>` emit executable bash scripts with targeted `<engine> apply -target='...'` and `<engine> apply -refresh-only -target='...'` recipes, plus CloudTrail lookup commands for AWS resources.

## Evidence

* Focused validation passed with `rtk go test ./...` (79 passed in 2 packages).
* Executable manual verification on `examples/drift-new-resource`:
  * Verified markdown output displays `null` -> `{"purpose":"produce a deterministic PLANNED status for tf-drift examples"}`.
  * Verified `-export-remediation` creates valid executable bash scripts with `pushd`, target commands, and `popd`.
  * Verified `-update-baseline` generates a clean JSON baseline without warnings.
  * Verified `-baseline` suppresses acknowledged changes, returning `exit code 0`.

## Reusable Pattern

When building CLI tools that audit external systems:
1. Never write temporary runtime binaries into the user's source code tree.
2. Separate known technical debt (baselines) from new regressions so CI adoption is friction-free.
3. Don't just alert that something changed—show the exact value diff and the deterministic command to fix or accept it.

## Limits

Remediation recipes provide targeted CLI commands; they do not automatically execute destructive applies or rewrite complex HCL modules, preserving human review in the loop.
