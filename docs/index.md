---
date: 2026-06-22
---

# Documentation Index

## Project

| Document | Description |
| --- | --- |
| [Changelog](https://github.com/brunobrise/tf-drift/blob/main/CHANGELOG.md) | Tracks notable user-facing changes by release. |
| [Contributing](https://github.com/brunobrise/tf-drift/blob/main/CONTRIBUTING.md) | Explains local development, test expectations, and contribution rules. |
| [Security Policy](https://github.com/brunobrise/tf-drift/security/policy) | Explains supported versions and private vulnerability reporting. |

## Specs

| Document | Description |
| --- | --- |
| [CLI Drift Detection Tool](./specs/019c0a6b28007abc-tf-drift-cli.md) | Defines the main CLI architecture, scan modes, drift classification, TUI behavior, and decision log. |
| [Example Terraform Status Fixtures](./specs/2026061720401300-examples-status-fixtures.md) | Defines the local Terraform examples used to demonstrate clean, planned, and error statuses. |
| [Scheduled Release Automation](./specs/019ed76d55fe0f64-scheduled-release-automation.md) | Defines daily release automation, SemVer tag selection, GoReleaser behavior, and release edge cases. |
| [Terraform Config Selection](./specs/2026061720540700-config-selection.md) | Defines interactive checkbox selection and CLI include/exclude filters for discovered configs. |
| [TUI Style System](./specs/019ee1b82f880df4-tui-style-system.md) | Defines selectable modern, classic, minimal, and accessible TUI styles for interactive views. |

## Stories

| Document | Description |
| --- | --- |
| [Config Selection Workflow](./stories/2026061720540700-config-selection.md) | Workflow story for narrowing detected configs before the worker pool runs. |
| [CI Annotations and Rule Predicates Success](./stories/019ef057c4ac7943-ci-annotations-and-rule-predicates.md) | Success story for SARIF output, predicate-based severity rules, OpenTofu fixtures, and monorepo docs. |
| [Drift Classification Success](./stories/019eed3e7df17cba-drift-classification.md) | Success story for separating external drift from normal pending Terraform/OpenTofu plan changes. |
| [Example Status Fixtures Workflow](./stories/2026061720401300-examples-status-fixtures.md) | Workflow story for providerless examples that validate mixed-status reporting. |
| [Home Path Display Success](./stories/0000019ee1c8ee3b-home-path-display.md) | Success story for shortening home-directory paths with `~` in human-facing output. |
| [OpenTofu Engine Selection Success](./stories/0000019ee1c12d56-opentofu-engine-selection.md) | Success story for resolving Terraform or OpenTofu once before scanning layers. |
| [Scheduled Release Automation Workflow](./stories/019ed76d55fe0f64-scheduled-release-automation.md) | Workflow story for daily release automation and the GitHub Actions failure modes it avoids. |
| [TUI Style System Success](./stories/019ee1b82f880df4-tui-style-system.md) | Success story for centralizing TUI styling while preserving readable status labels and current controls. |
| [Version Reporting Success](./stories/019ee1cb66012699-version-reporting.md) | Success story for consistent release and source-build version output through `-version` and `-v`. |
