# tf-drift

<p align="center">
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go"></a>
  <a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-indigo?style=for-the-badge&logo=appveyor" alt="Bubble Tea TUI"></a>
  <a href="https://www.terraform.io"><img src="https://img.shields.io/badge/Terraform-7B42BC?style=for-the-badge&logo=terraform&logoColor=white" alt="Terraform"></a>
</p>

`tf-drift` is a Go utility to detect, filter, and inspect configuration drift across multi-layered Terraform workspaces concurrently. It features an interactive, height-adaptive TUI and a non-interactive mode for CI/CD.

## Quick Start

```bash
# Clone and build
git clone https://github.com/brunobrise/tf-drift.git
cd tf-drift
make build

# Install globally
go build -o ~/.local/bin/tf-drift

# Run interactive scan
tf-drift -dir ../your-infrastructure-dir
```

## CLI Flags

| Flag | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `-dir` | string | `.` | Directory to scan. |
| `-env` | string | `""` | Filter layers by environment folder name. |
| `-layer` | string | `""` | Target a specific layer path. |
| `-concurrency` | int | `5` | Max concurrent plan execution workers. |
| `-format` | string | `text` | Non-interactive output format (`text`, `json`, `markdown`, `slack`). |
| `-lock` | bool | `false` | Enable state locking. |
| `-rules` | string | `rules.json` | Path to rules configuration. |
| `-non-interactive` | bool | `false` | Disable TUI mode. |
| `-profile-override` | string | `""` | Override AWS profile and comment out `assume_role`. |
| `-local-profile` | bool | `false` | Comment out `assume_role` and uncomment existing profiles. |

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
