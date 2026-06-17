---
date: 2026-06-17
type: workflow
status: validated
related_specs:
  - ../specs/019ed76d55fe0f64-scheduled-release-automation.md
---

# Scheduled Release Automation Workflow

## Context

`tf-drift` already had a tag-triggered GoReleaser workflow that builds binaries and updates the Homebrew tap. The missing piece was an automatic release check so merged commits do not wait on a manual tag.

## Intended Outcome

Every day at midnight UTC, the repository should publish a release only when the default branch has commits after the latest stable release tag. Manual tag releases and manual workflow dispatch should remain available.

## Failure Modes Avoided

- A separate workflow that creates tags with `GITHUB_TOKEN` would not trigger the tag-push release workflow.
- A top-of-hour schedule can be delayed by GitHub Actions load, so manual dispatch remains documented as recovery.
- A release without tests could publish broken binaries, so the release job now runs the race test suite before GoReleaser.
- Parallel release attempts could collide on tags or uploaded assets, so the workflow serializes release runs.

## Evidence

Validation used `go test -race -v ./...`, `actionlint`, `git diff --check`, and a local SemVer simulation. The current tag history would compute `v1.1.0` from unreleased `feat:` commits after `v1.0.0`.

## Reusable Pattern

For GitHub Actions release automation, keep tag calculation and publishing in the same workflow when using `GITHUB_TOKEN`. Use a local release tag plus GoReleaser `target_commitish` instead of depending on a second workflow run.

## Limits

The Homebrew formula path still uses GoReleaser's deprecated `brews` section. That preserves the existing `brew install tf-drift` install flow, but it should be revisited before moving to a future GoReleaser major version that removes formula support.
