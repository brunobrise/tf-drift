# tf-drift

`tf-drift` is a high-performance, Go-based CLI tool designed to detect, filter, and inspect configuration drift across multi-layered Terraform workspaces/layers in parallel.

It features a modern, interactive Terminal User Interface (TUI) built on Charm CLI's Bubble Tea, as well as a non-interactive reporting mode designed for CI/CD pipelines.

## Features

- **Recursive Scanning:** Automatically discovers directories containing `.tf` files and backend configurations.
- **Worker Pool Concurrency:** Runs parallel scans with a bounded concurrency pool (`-concurrency`) to maximize speed and prevent cloud API throttling.
- **State-Safe by Default:** Defaults to `-lock=false` (passive scan) to prevent blocking active deployment pipelines.
- **Signal Handled Cancellation:** Gracefully intercepts termination signals (`Ctrl+C` / `SIGINT`) to allow child Terraform processes to exit cleanly.
- **Smart Ignore and Severity Rules:** Uses a `rules.json` file to filter out tag noise, autoscaling desired capacity updates, or layer-specific metadata, while flagging critical changes (e.g., IAM, Security Groups) with custom severities.
- **Modern Interactive TUI:** Scrollable list of layers, live progress spinner, interactive status filters, and detailed drift inspection view.
- **Multi-Format Fallback Reporters:** Auto-falls back in non-TTY/CI environments to report in JSON, Markdown (suitable for PR comments), Slack messaging blocks, or plain text.

---

## Installation

Ensure you have Go installed (>= 1.25). Clone the repository and compile the binary:

```bash
cd ~/Code/brunobrise/tf-drift
go build -o tf-drift
```

---

## CLI Usage

Run the tool in the current directory or target a specific repository path:

```bash
# Start the interactive TUI in the current directory
./tf-drift

# Scan a target repository
./tf-drift -dir ../your-infrastructure-dir

# Scan only development workload layers
./tf-drift -dir ../your-infrastructure-dir -env workload_api_dev

# Scan a single specific layer path
./tf-drift -dir ../your-infrastructure-dir -layer aws/workload_api_dev/007_secret_dev
```

### Options

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-dir` | `.` | Target directory to recursively scan. |
| `-env` | `""` | Filter layers by environment folder name (e.g., `workload_api_dev`). |
| `-layer` | `""` | Filter scan to a specific relative layer path. |
| `-concurrency` | `5` | Maximum concurrent plan runners. |
| `-format` | `text` | Non-interactive output format (`text`, `json`, `markdown`, `slack`). |
| `-lock` | `false` | Enable state locking during plan execution. |
| `-rules` | `rules.json` | Path to rules configuration file. |
| `-non-interactive` | `false` | Force disable TUI and output logs to stdout. |

---

## Rules Configuration (`rules.json`)

Configure ignore rules and resource severity rankings to filter out noise:

```json
{
  "global_ignores": {
    "resource_types": ["aws_autoscaling_group"],
    "attributes": ["tags", "desired_capacity", "lifecycle"]
  },
  "severity_classification": {
    "aws_iam_policy": "CRITICAL",
    "aws_security_group_rule": "CRITICAL",
    "aws_rds_cluster": "HIGH",
    "aws_route53_record": "MEDIUM"
  },
  "layer_ignores": {
    "aws/workload_api_dev/500_rds_dev": {
      "attributes": ["database_name"]
    }
  }
}
```

---

## CI/CD Exit Codes

When run non-interactively (e.g., in GitHub Actions), the tool returns:
- **`0`**: Clean (no configuration drift found).
- **`1`**: Execution error (failed to run plan, syntax error, or missing credentials).
- **`2`**: Drift detected (unignored configuration changes are present in real infrastructure).
