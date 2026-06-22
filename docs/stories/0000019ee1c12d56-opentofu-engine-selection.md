---
date: 2026-06-22
type: success
status: validated
related_specs:
  - ../specs/019c0a6b28007abc-tf-drift-cli.md
---

# OpenTofu Engine Selection

## Context

Terraform and OpenTofu users need the same drift workflow: discover configs, initialize each layer, run a detailed-exitcode plan, then parse `show -json`.

## Intended Outcome

`tf-drift` should run against OpenTofu without forcing existing Terraform users to change their setup. The default engine is `auto`, preferring `tofu` when installed and falling back to `terraform`.

## What Should Work

OpenTofu keeps Terraform-compatible command shapes for `init`, `plan -detailed-exitcode`, and `show -json`, so the runner can share one execution path after resolving the binary.

## Failure Modes To Surface

OpenTofu migration failures usually come from provider or module resolution differences, omitted provider source addresses, provider version jumps, encrypted state or plan configuration, and saved plans containing sensitive data.

## Reusable Pattern

Resolve the engine once before workers start, then pass the resolved executable into the worker pool. This keeps status reporting deterministic and prevents different workers from choosing different binaries.

## Evidence

Validation covers engine resolution, worker propagation, CLI flag behavior, runner JSON parsing, and OpenTofu-specific plan JSON fixtures for deposed replacement drift, provider names under `registry.opentofu.org`, import metadata, unknown values, and read-only data source changes that should not count as drift.

## Limits

`tf-drift` does not migrate configurations between Terraform and OpenTofu. It only selects and runs an installed compatible executable.
