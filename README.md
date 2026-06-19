# tf-drift

<p align="center">
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go"></a>
  <a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-indigo?style=for-the-badge&logo=appveyor" alt="Bubble Tea TUI"></a>
  <a href="https://www.terraform.io"><img src="https://img.shields.io/badge/Terraform-7B42BC?style=for-the-badge&logo=terraform&logoColor=white" alt="Terraform"></a>
  <a href="https://opentofu.org"><img src="https://img.shields.io/badge/OpenTofu-FFDA18?style=for-the-badge" alt="OpenTofu"></a>
</p>

`tf-drift` is a Go utility to detect, filter, and inspect configuration drift across multi-layered Terraform and OpenTofu workspaces concurrently. It features an interactive, height-adaptive TUI and a non-interactive mode for CI/CD.

## Installation

### Via Homebrew (Recommended)

```bash
# Tap the custom repository
brew tap brunobrise/homebrew-tap

# Trust the tap (Required for Homebrew 6.0+)
brew trust brunobrise/homebrew-tap

# Install the utility
brew install tf-drift
```

### From Source

```bash
# Clone and build
git clone https://github.com/brunobrise/tf-drift.git
cd tf-drift
make build

# Install globally
go build -o ~/.local/bin/tf-drift
```

## Quick Start

```bash
# Run interactive scan on your infrastructure directory
tf-drift -dir ../your-infrastructure-dir
```

In interactive mode, `tf-drift` first shows a checkbox picker for discovered Terraform/OpenTofu configs. Use `Space` to tick or untick a config, `a` to select all, `n` to select none, and `Enter` to scan the selected configs.

By default, `tf-drift` uses `-engine auto`: it runs OpenTofu (`tofu`) when installed, otherwise it falls back to Terraform (`terraform`). Pin the engine in CI when deterministic binary selection matters:

```bash
tf-drift -dir ../your-infrastructure-dir -engine opentofu
tf-drift -dir ../your-infrastructure-dir -engine terraform
```

## Examples

The repository includes local Terraform examples for the main scan statuses:

```bash
tf-drift -dir examples -non-interactive || true
tf-drift -dir examples -non-interactive -format json || true
tf-drift -dir "examples/{clean-empty|drift-new-resource}" -non-interactive
tf-drift -dir examples -non-interactive -include "clean-empty,drift-*" || true
tf-drift -dir examples -non-interactive -exclude "error-*"
```

See `examples/README.md` for the expected `CLEAN`, `DRIFTED`, and `ERROR` layers.

## CLI Flags

| Flag | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `-dir` | string | `.` | Directory to scan. |
| `-env` | string | `""` | Filter layers by environment folder name. |
| `-layer` | string | `""` | Target a specific layer path. |
| `-include` | string | `""` | Comma-separated config suffix or glob patterns to include. |
| `-exclude` | string | `""` | Comma-separated config suffix or glob patterns to exclude. |
| `-concurrency` | int | `5` | Max concurrent plan execution workers. |
| `-format` | string | `text` | Non-interactive output format (`text`, `json`, `markdown`, `slack`). |
| `-lock` | bool | `false` | Enable state locking. |
| `-rules` | string | `rules.json` | Path to rules configuration. |
| `-non-interactive` | bool | `false` | Disable TUI mode. |
| `-tui-style` | string | `modern` | Interactive TUI style (`modern`, `classic`, `minimal`, `accessible`). Can also be set with `TF_DRIFT_TUI_STYLE`; `NO_COLOR` forces `minimal`. |
| `-profile-override` | string | `""` | Override AWS profile and comment out `assume_role`. |
| `-local-profile` | bool | `false` | Comment out `assume_role` and uncomment existing profiles. |
| `-engine` | string | `auto` | IaC engine to run (`auto`, `terraform`, `opentofu`, `tofu`). |
| `-reconfigure` | bool | `false` | Run engine `init` with `-reconfigure`. |
| `-migrate-state` | bool | `false` | Run engine `init` with `-migrate-state`. |

Selection filters run after `-dir`, `-env`, and `-layer`. Include filters run before exclude filters and preserve discovery order.

## Rules Configuration (`rules.json`)

```json
{
  "global_ignores": {
    "resource_types": ["aws_autoscaling_group"],
    "attributes": ["tags", "desired_capacity"]
  },
  "severity_classification": {
    "aws_iam_policy": "CRITICAL",
    "aws_rds_cluster": "HIGH"
  }
}
```

## Diagnostics & Exit Codes

* **Exit Codes**: `0` (clean), `1` (failure), `2` (drift detected).
* **Logs**: Captured in `tf-drift.log` in TUI mode to prevent screen corruption, or printed to `Stderr` in non-interactive mode.

## OpenTofu Notes

OpenTofu support uses the same execution flow as Terraform: `init`, `plan -detailed-exitcode`, and `show -json`. For OpenTofu migration failures, check explicit provider source addresses, registry resolution, provider version constraints, state encryption keys, and saved plan handling. Non-interactive runs set `TF_IN_AUTOMATION=1` for child engine commands.

## Release Automation

Releases are built with GoReleaser and published to GitHub Releases and the Homebrew tap. The release workflow runs daily at midnight UTC and only publishes when new commits exist after the latest stable `vX.Y.Z` tag. See `SETUP.md` for release credentials, manual release options, and recovery steps.
