---
date: 2026-06-22
type: success
status: validated
related_specs:
  - ../specs/019c0a6b28007abc-tf-drift-cli.md
---

# CI Annotations and Rule Predicates Success

## Context

The roadmap called for CI annotations, clearer non-interactive automation output, more OpenTofu plan JSON fixture coverage, severity classification predicates, and better large monorepo guidance.

## Bet

SARIF can serve GitHub code scanning and generic CI annotation workflows without adding a GitHub-only API integration. Ordered severity predicates can keep the simple resource-type map while allowing teams to escalate only the changes that matter in production layers.

## What Worked

`-format sarif` emits SARIF 2.1.0 with stable `tf-drift.external-drift`, `tf-drift.planned-change`, and `tf-drift.execution-error` rule IDs. `severity_rules` add predicates for resource type, changed attributes, plan actions, classification, layer glob, and address glob while preserving `severity_classification` fallback behavior.

OpenTofu fixtures now cover deposed replacement drift, OpenTofu provider source names, import metadata, unknown values, and read-only data source changes. README and the CLI spec document large monorepo scans with brace/glob `-dir` patterns, include/exclude filters, `-mode drift`, and SARIF output.

## Evidence

Focused validation passed with `rtk go test ./internal/drift`. Tests cover SARIF schema shape and severity-to-level mapping, predicate rule matching, legacy severity fallback, and OpenTofu fixture parsing.

## Reusable Pattern

Prefer portable report formats before vendor-specific API integrations. Keep old config keys working, then add ordered predicate rules for users who need precision without custom code.

## Limits

SARIF locations point at the scanned layer directory, not at individual Terraform source lines. The plan JSON does not include stable source-line locations for every drifted resource.
