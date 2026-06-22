# Contributing to tf-drift

Thank you for your interest in contributing to `tf-drift`! We welcome focused community contributions that improve drift detection, Terraform/OpenTofu compatibility, reporting, documentation, and release quality.

---

## 1. Getting Started

### Prerequisites
* Go (version `>= 1.24`)
* Terraform (version `>= 1.0` recommended)
* `make` (optional, for automation)

### Local Environment Setup
1. Fork and clone the repository:
   ```bash
   git clone https://github.com/brunobrise/tf-drift.git
   cd tf-drift
   ```
2. Download dependencies:
   ```bash
   go mod download
   ```

---

## 2. Development & Code Conventions

### Repository Structure
- **`/cmd/tf-drift/main.go`**: Command entry point. Keep the CLI setup and flag handling here.
- **`/internal/drift/`**: Internal package containing the core logic (plan execution, TUI, rule evaluation). All business logic belongs here.

### Coding Style & Formatting
We follow standard Go coding conventions. Always format your code before committing:
```bash
go fmt ./...
```

### Maximum File Size
To maintain codebase readability, please keep Go files under **420 lines of code** (excluding tests).

### Scope
Keep pull requests small and reviewable. Large feature work should start as an issue that describes the use case, expected CLI behavior, and any Terraform/OpenTofu compatibility concerns.

---

## 3. Running Tests

We prioritize high test coverage. Ensure all unit tests pass locally before proposing changes:

```bash
# Run tests recursively with the race detector
go test -race -v ./...
```

For CLI or TUI changes, also run at least one manual scan against `examples/`:

```bash
go run ./cmd/tf-drift -dir examples -non-interactive || true
```

---

## 4. Git Commit Guidelines

To maintain a clean and structured commit history, we enforce **single-line Conventional Commits**.

### Commit Message Format
Every commit message must fit on a **single line** and follow this pattern:
```
<type>: <description>
```

### Allowed Types
* `feat`: Introduce a new feature (e.g. `feat: add credentials scanning`).
* `fix`: Fix a bug (e.g. `fix: resolve crash on missing rules file`).
* `refactor`: Restructure code without changing behavior (e.g. `refactor: clean up worker queue`).
* `docs`: Update documentation (e.g. `docs: update setup guide`).
* `test`: Add or modify tests (e.g. `test: cover detail view navigation`).
* `chore`: Maintain build configs, rules, or dependencies (e.g. `chore: update dependencies`).

### Constraints
* **DO NOT** use `git add .` or similar commands to stage everything blindly. Only stage files you explicitly modified.
* **DO NOT** commit with a multi-line body, co-authors, or "Co-Authored-By" tags.
* **DO NOT** use `--no-verify`.

---

## 5. Maintainer Expectations

Maintainers review issues and pull requests as capacity allows. Security reports should follow `SECURITY.md`; please do not disclose vulnerabilities in public issues.
