# tf-drift Examples

These examples exercise the main `tf-drift` statuses without cloud credentials, remote state, or external providers. They can run with Terraform or OpenTofu.

## Layout

| Directory | Expected status | Purpose |
| --- | --- | --- |
| `clean-empty` | `CLEAN` | Valid Terraform/OpenTofu config with no managed resources. |
| `drift-new-resource` | `PLANNED` | Valid Terraform/OpenTofu config with a built-in `terraform_data` resource that has not been applied. |
| `error-invalid-config` | `ERROR` | Intentionally invalid Terraform/OpenTofu config for error reporting. |

## Run All Examples

```bash
tf-drift -dir examples -non-interactive
```

The command exits with code `1` because one layer is intentionally invalid. To inspect output while ignoring the expected error exit code:

```bash
tf-drift -dir examples -non-interactive || true
```

## JSON Output

```bash
tf-drift -dir examples -non-interactive -format json || true
```

## Scan Selected Examples

```bash
tf-drift -dir "examples/{clean-empty|drift-new-resource}" -non-interactive
```

The selected scan exits with code `2` because `drift-new-resource` intentionally plans a new resource. Use `-mode drift` if you want to ignore ordinary pending config changes and report only external drift.
