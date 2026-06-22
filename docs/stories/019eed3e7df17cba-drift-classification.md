---
date: 2026-06-22
type: success
status: validated
related_specs:
  - ../specs/019c0a6b28007abc-tf-drift-cli.md
  - ../specs/2026061720401300-examples-status-fixtures.md
---

# Drift Classification Success

## Context

`tf-drift` previously treated every non-no-op Terraform/OpenTofu plan delta as drift. That made unapplied configuration changes look the same as external infrastructure changes, even though plan JSON exposes `resource_drift` separately from `resource_changes`.

## Bet

Classifying structured plan JSON can preserve the existing CI exit-code sensitivity while making reports more accurate for humans and automation.

## What Worked

The runner now reads `resource_drift` as `EXTERNAL_DRIFT` and normal `resource_changes` as `PLANNED_CHANGE`. The `-mode both|drift|plan` flag lets CI choose whether planned config deltas should count for the run, while default `both` keeps the old "any selected change exits 2" behavior.

Reports, JSON, Slack text, and the TUI all show the classification. Example fixtures now document an unapplied `terraform_data` resource as `PLANNED` instead of drift.

## Evidence

Focused parser, report, and TUI tests cover drift-only, plan-only, mixed input, mode filtering, invalid mode, and JSON classification. Full verification passed with `rtk go test ./...` and `rtk go test -race ./...`. A temp-copy example smoke showed default `both` exits `2` for the planned fixture, while `-mode drift` ignores that planned-only change and exits `0`.

## Reusable Pattern

Prefer Terraform/OpenTofu plan JSON fields over human output. Preserve old CI contracts through defaults, then add explicit modes or fields so users can opt into narrower behavior without losing visibility.

## Limits

If the same resource address appears in both `resource_drift` and `resource_changes`, external drift wins to avoid double-counting drift remediation as an ordinary planned change.
