---
date: 2026-06-17
type: workflow
status: validated
related_specs:
  - ../specs/2026061720401300-examples-status-fixtures.md
---

# Example Status Fixtures Workflow

## Context

Developers need a local way to verify the scanner's mixed-status reporting without provisioning cloud resources or editing live Terraform state.

## Intended Outcome

The repository should contain small, runnable Terraform examples that demonstrate clean, drifted, and error states through the same CLI path users run in CI or locally.

## Reusable Pattern

Prefer providerless or built-in Terraform fixtures for CLI examples. This keeps examples deterministic, avoids credential setup, and still exercises discovery, initialization, planning, and reporting.

## Evidence

Validation is based on focused discovery tests, the full Go test suite, and non-interactive `tf-drift` runs against `examples/`.

## Limits

These fixtures prove status reporting, not cloud provider behavior or remote backend behavior.
