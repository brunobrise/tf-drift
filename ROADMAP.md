# Roadmap

`tf-drift` focuses on fast, understandable Terraform/OpenTofu drift detection for layered infrastructure repositories.

## Current Focus

* Keep external drift classification distinct from pending config changes.
* Improve non-interactive output for CI, Slack, and release automation.
* Keep the TUI readable across terminal sizes and accessibility modes.
* Preserve deterministic engine selection for Terraform and OpenTofu.

## Candidate Work

* SARIF or GitHub Checks output for CI annotations.
* More fixture coverage for OpenTofu-specific plan JSON edge cases.
* Additional rule predicates for severity classification.
* Better documentation for large monorepo layouts.

## Out of Scope

* Managing infrastructure state directly.
* Applying Terraform/OpenTofu plans.
* Replacing Terraform Cloud, OpenTofu workflows, or policy engines.
