# CI/CD & Release Setup Guide

This guide explains how to configure, trigger, and maintain the CI/CD pipeline for `tf-drift`, including automated tests, security audits, dependency management, cross-platform builds, and Homebrew tap updates.

---

## 1. CI Workflow (Continuous Integration)

The Continuous Integration workflow validates all code changes before they are merged into the default branch.

### Triggers
* **Pushes** to the `main` branch.
* **Pull Requests** targeting any branch.

### Jobs & Checks
1. **Code Quality & Security Lints (`golangci-lint`)**:
   * Uses `golangci-lint` to run static code analysis, ensuring style alignment and identifying code smell.
2. **Vulnerability Audit (`govulncheck`)**:
   * Scans dependencies for known security vulnerabilities.
   * Runs using **Go 1.25** to satisfy the tool's runtime dependencies.
3. **Unit Tests & Race Detection**:
   * Executes the Go test suite recursively using the `-race` detector to identify concurrency issues.
   * Runs using **Go 1.24** to guarantee compatibility with the target runtime environment.

---

## 2. Dependency Management (Dependabot)

Dependabot is configured to keep dependencies secure and up to date automatically.

* **Ecosystems**: Go modules (`gomod`) and GitHub Actions (`github-actions`).
* **Schedule**: Checked weekly.
* **Grouping**: All updates are automatically consolidated into single PRs (`go-dependencies` and `github-actions-dependencies`) to minimize notification and pull request noise.

---

## 3. Releases & GoReleaser (Continuous Delivery)

Releases are fully automated and driven by Git tags.

### Step 3.1: Configure Homebrew Tap Credentials
Because GitHub Actions' default `GITHUB_TOKEN` is scoped to the current repository, it cannot write to a separate Homebrew tap repository (e.g. `github.com/brunobrise/homebrew-tap`). You must provide a custom token with write access.

#### Part A: Generate a Fine-grained GitHub PAT
1. Log in to GitHub.
2. Go to **Settings** -> **Developer settings** -> **Personal access tokens** -> **Fine-grained tokens**.
3. Click **Generate new token**.
4. Configure the token:
   * **Token name**: `Homebrew Tap Push Token`
   * **Expiration**: E.g., `90 days`
   * **Repository access**: Select **Only select repositories** -> Select your **`homebrew-tap`** repository.
   * **Permissions**: Under **Repository permissions**, find **Contents** and select **Read and write**.
5. Click **Generate token and copy it**.

#### Part B: Add the Token as a Repository Secret
1. Go to your **`tf-drift`** repository on GitHub.
2. Click **Settings** -> **Secrets and variables** -> **Actions**.
3. Click **New repository secret**.
4. Configure:
   * **Name**: `HOMEBREW_TAP_TOKEN`
   * **Secret**: Paste the copied token value.
5. Click **Add secret**.

### Step 3.2: Tag and Publish a Release
1. Create a Semantic Version tag pointing to the release commit:
   ```bash
   git tag v1.0.0
   ```
2. Push the tag to GitHub to trigger the release workflow:
   ```bash
   git push origin v1.0.0
   ```

### Automated Release Actions
1. Runs the lint and test suites.
2. Builds cross-platform binaries using **GoReleaser**:
   * **macOS** (`amd64` / `arm64`)
   * **Linux** (`amd64` / `arm64`)
   * **Windows** (`amd64`)
3. Publishes a GitHub Release with compiled archives, checksums, and auto-generated changelogs.
4. Automatically commits the updated formula Ruby file directly to `brunobrise/homebrew-tap`.

---

## 4. Installing via Homebrew Tap

Once the release pipeline finishes, users can install `tf-drift` via Homebrew:

```bash
# Add the custom Homebrew tap
brew tap brunobrise/homebrew-tap

# Install tf-drift
brew install tf-drift

# Verify installation
tf-drift -version
```
