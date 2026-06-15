# CI/CD & Release Setup Guide

This guide explains how to configure and trigger the CI/CD pipeline for `tf-drift`, including automated cross-platform builds and Homebrew tap updates.

---

## 1. Initial Setup: Configure GitHub Secrets

Because GitHub Actions’ default `GITHUB_TOKEN` is scoped to the current repository, it cannot write to a separate Homebrew tap repository (e.g. `github.com/brunobrise/homebrew-tap`). You must provide a custom token with access to your tap.

### Step 1.1: Generate a Fine-grained GitHub PAT
1. Log in to GitHub.
2. Click your profile photo in the top-right corner -> **Settings**.
3. Scroll down the left sidebar and click **Developer settings**.
4. In the left sidebar, click **Personal access tokens** -> **Fine-grained tokens**.
5. Click **Generate new token**.
6. Configure the token:
   * **Token name**: `Homebrew Tap Push Token`
   * **Expiration**: E.g., `90 days`
   * **Repository access**: Select **Only select repositories** -> Select your **`homebrew-tap`** repository.
   * **Permissions**: Under **Repository permissions**, find **Contents** and select **Read and write**.
7. Click **Generate token** and copy the generated token immediately (you will not be able to see it again).

### Step 1.2: Add Token as a Repository Secret
1. Go to your **`tf-drift`** repository on GitHub.
2. Click **Settings** (the gear icon on the tab bar).
3. In the left sidebar, click **Secrets and variables** -> **Actions**.
4. Click **New repository secret**.
5. Configure the secret:
   * **Name**: `HOMEBREW_TAP_TOKEN`
   * **Secret**: Paste the copied token value.
6. Click **Add secret**.

---

## 2. Triggering a New Release

Releases are fully automated and driven by git tags.

### Step 2.1: Tag the Commit
Create a Semantic Version tag pointing to the commit you want to release:
```bash
git tag v1.0.0
```

### Step 2.2: Push the Tag
Push the tag to GitHub. This triggers the release workflow:
```bash
git push origin v1.0.0
```

### What happens automatically:
1. **CI Pipeline** runs tests and audits.
2. **GoReleaser** builds binaries for:
   * **macOS** (`amd64` / `arm64`)
   * **Linux** (`amd64` / `arm64`)
   * **Windows** (`amd64`)
3. A **GitHub Release** is created containing the compiled binaries, checksums, and auto-generated changelogs.
4. The **Homebrew Formula** Ruby file in `brunobrise/homebrew-tap` is automatically updated with the new tag url and SHA256 checksums.

---

## 3. Installing via Homebrew Tap

Once GoReleaser has successfully updated the tap, users can install `tf-drift` via Homebrew:

```bash
# Add your custom Homebrew tap
brew tap brunobrise/tap

# Install tf-drift
brew install tf-drift

# Verify installation
tf-drift -version
```
