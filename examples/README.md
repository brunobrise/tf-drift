# tf-drift Examples

These examples exercise the main `tf-drift` statuses without cloud credentials, remote state, or external providers.

## Layout

| Directory | Expected status | Purpose |
| --- | --- | --- |
| `clean-empty` | `CLEAN` | Valid Terraform config with no managed resources. |
| `drift-new-resource` | `DRIFTED` | Valid Terraform config with a built-in `terraform_data` resource that has not been applied. |
| `error-invalid-config` | `ERROR` | Intentionally invalid Terraform config for error reporting. |

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

The selected scan exits with code `2` because `drift-new-resource` intentionally plans a new resource.
