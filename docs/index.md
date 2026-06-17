---
date: 2026-06-17
---

# Documentation Index

## Specs

| Document | Description |
| --- | --- |
| [CLI Drift Detection Tool](./specs/019c0a6b28007abc-tf-drift-cli.md) | Defines the main CLI architecture, workflow, TUI behavior, and decision log. |
| [Example Terraform Status Fixtures](./specs/2026061720401300-examples-status-fixtures.md) | Defines the local Terraform examples used to demonstrate clean, drifted, and error statuses. |
| [Terraform Config Selection](./specs/2026061720540700-config-selection.md) | Defines interactive checkbox selection and CLI include/exclude filters for discovered configs. |

## Stories

| Document | Description |
| --- | --- |
| [Config Selection Workflow](./stories/2026061720540700-config-selection.md) | Workflow story for narrowing detected configs before the worker pool runs. |
| [Example Status Fixtures Workflow](./stories/2026061720401300-examples-status-fixtures.md) | Workflow story for providerless examples that validate mixed-status reporting. |
