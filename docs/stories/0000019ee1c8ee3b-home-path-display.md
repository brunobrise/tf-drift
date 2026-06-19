---
date: 2026-06-19
type: success
status: draft
related_specs:
  - ../specs/019c0a6b28007abc-tf-drift-cli.md
---

# Home Path Display

## Context

Layer paths can expose long local home-directory prefixes in the TUI and reports, especially inside desktop app output.

## Intended Outcome

Human-facing output should display `~` instead of `/Users/<name>`, `/home/<name>`, or `/root` when a layer path is under the current user's home directory.

## What Worked

The display helper only changes presentation strings. Raw paths remain unchanged for discovery, execution, rule matching, worker results, and JSON output.

## Evidence

Focused tests cover exact home paths, child paths, sibling-prefix safety, root home paths, picker fallback display, and text reports.

## Limits

JSON report paths remain raw to avoid breaking machine-readable integrations.
