# Copilot Sync (`cops`)

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev/)

*The deterministic package manager for your GitHub Copilot agent files.*

It's time to share your knowledge.

---

## 💡 Core Principles

### 🎯 Deterministic Approach
`cops` manages your agent assets through a `copilot.toml` manifest file, allowing you to pin every asset to a defined version — a release tag, branch, or commit hash.

### 💻 Dev-Centered
Built as a blazing-fast, zero-dependency Go binary. `cops` gets out of your way.

### 🌳 Git-Based Source of Truth
No proprietary registries to learn. `cops` pulls directly from your existing GitHub repositories. Pin your team's DDD practices to a specific release tag (`@v1.2.0`), a branch (`@main`), or a commit hash. If it lives in Git, `cops` can sync it.

---

## ⚔️ Why use `cops`?

### Share your assets at scale

One of the main pain-points I encountered in my teams was the ability to easily share and maintain instructions, agents and prompts across multiple projects and teams. `cops` aims to provide the simplest way to deal with it.

### Avoid agent injections

By pinning agent files to dedicated versions, you limit the risk of unattended behaviors from self-updated skills.

### Copilot-centered

Share not only `skills` but also `instructions`, `prompts`, and `agents` — the four asset types that GitHub Copilot supports.

---

## 📦 Installation

### Homebrew (macOS)

```bash
brew install --cask cbout22/tap/cops
```

### APT / Debian

Download the `.deb` package from the [latest release](https://github.com/cbout22/copilot-sync/releases/latest) and install it:

```bash
curl -LO https://github.com/cbout22/copilot-sync/releases/latest/download/cops_<version>_linux_amd64.deb
sudo dpkg -i cops_<version>_linux_amd64.deb
```

### Scoop (Windows)

```powershell
scoop bucket add cops https://github.com/cbout22/scoop-bucket
scoop install cops
```

### From source (Go)

```bash
go install github.com/cbout22/copilot-sync/cmd/cops@latest
```

### Build locally

```bash
git clone https://github.com/cbout22/copilot-sync.git
cd copilot-sync
go build -ldflags "-X github.com/cbout22/copilot-sync/internal/cli.version=$(git describe --tags --always)" -o cops ./cmd/cops/
```

The resulting `cops` binary can be placed anywhere on your `$PATH`.

### Verify

```bash
cops --version
```

---

## 🚀 Quick Start

**1. open any github repository**

**2. fetch a remote file from github/awesome-copilot-agents**
```bash
$ cops instructions use reviews github/awesome-copilot/instructions/code-review-generic.instructions.md@latest 
📦 Adding instructions/reviews from github/awesome-copilot/instructions/code-review-generic.instructions.md@latest...
✅ instructions/reviews synced to .github/instructions/reviews.instructions.md
```

**3. That's it. Your Copilot agent files are now version-controlled and reproducible.**

Your `copilot.toml` manifest now contains the new entry:
```bash
✗ cat copilot.toml 
[instructions]
  reviews = "github/awesome-copilot/instructions/code-review-generic.instructions.md@latest"
```
And your lock file is updated:
```bash
✗ cat .cops.lock 
{
  "version": 1,
  "entries": {
    "instructions/reviews": {
      [...]
    }
  }
}
```


---

## 📖 CLI Reference

### Command Tree

```
cops
├── instructions              # Manage instruction files
│   ├── use <name> <ref>      #   Add & download an instruction
│   └── unuse <name>          #   Remove an instruction
├── agents                    # Manage agent files
│   ├── use <name> <ref>      #   Add & download an agent
│   └── unuse <name>          #   Remove an agent
├── prompts                   # Manage prompt files
│   ├── use <name> <ref>      #   Add & download a prompt
│   └── unuse <name>          #   Remove a prompt
├── skills                    # Manage skill directories
│   ├── use <name> <ref>      #   Add & download a skill (directory)
│   └── unuse <name>          #   Remove a skill
├── sync                      # Download all assets from copilot.toml
├── check [--strict]          # Validate local state matches manifest
└── --version                 # Print version
```

---

### `cops <type> use`

Add an asset to `copilot.toml`, download it from GitHub.

```bash
cops <type> use <name> <org>/<repo>/path/to/file@<ref>
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `type` | One of `instructions`, `agents`, `prompts`, `skills` |
| `name` | Local name for the asset (used as filename) |
| `ref` | GitHub reference in the format `org/repo/path@version` |

**Reference format:**

The `@<ref>` suffix controls which version to fetch:

| Ref | Example | Description |
|-----|---------|-------------|
| `@latest` | `my-org/repo/file@latest` | Resolves to the repo's default branch |
| `@<tag>` | `my-org/repo/file@v1.2.0` | A specific Git tag |
| `@<branch>` | `my-org/repo/file@main` | A branch name |
| `@<sha>` | `my-org/repo/file@a1b2c3d` | A commit SHA |

**Examples:**

```bash
# Add an agent file
cops agents use reviewer my-org/copilot-agents/personas/senior-reviewer.md@main

# Pin an instruction to a release tag
cops instructions use clean-code my-org/standards/ddd/clean-code.md@v1.2

# Download a skill directory
cops skills use kubernetes my-org/mcp-tools/k8s-cluster-manager@latest
```

---

### `cops <type> unuse`

Remove an asset from `copilot.toml`, delete the local file/directory, and remove the lock file entry.

```bash
cops <type> unuse <name>
```

**Example:**

```bash
cops agents unuse reviewer
```

---

### `cops sync`

Download or update **all** assets declared in `copilot.toml`. This is the main command to keep your local files in sync with the manifest.

```bash
cops sync
```

**Behavior:**
- Iterates over every entry in `copilot.toml`
- Downloads (or re-downloads) each asset from GitHub
- Resolves `@latest` references to the current default branch
- Updates the `.cops.lock` file with resolved commit SHAs and checksums
- Reports ✅ or ❌ per entry

---

### `cops check`

Validate that all entries in `copilot.toml` have corresponding local files and matching lock file entries.

```bash
cops check [--strict]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--strict` | Exit with a non-zero code if any asset is missing or stale (useful for CI/CD) |

**Detects:**
- Assets that were never synced
- Missing local files (deleted since last sync)
- Entries not present in the lock file
- Ref mismatches between manifest and lock

---

## 📝 Configuration

### `copilot.toml`

The manifest file lives at the root of your project. It contains four optional sections, one per asset type:

```toml
# copilot.toml — Copilot Sync manifest
# Format: name = "org/repo/path/to/file@ref"

[instructions]
clean-code = "my-org/standards/practices/ddd/clean-code.md@v1.2"
security   = "my-org/standards/security/guidelines.md@main"

[agents]
reviewer     = "my-org/copilot-agents/personas/senior-reviewer.md@main"
security-bot = "my-org/copilot-agents/security/scanner.md@v2.0"

[prompts]
api-design = "my-org/prompts/api/restful-design.md@v1.0"
tests      = "my-org/prompts/testing/unit-tests.md@latest"

[skills]
kubernetes-mcp = "my-org/mcp-tools/k8s-cluster-manager@latest"
database       = "my-org/mcp-tools/db-manager@v3.1"
```

### Destination Mapping

Each asset type is downloaded to a specific directory under `.github/`:

| Section | Local Path | Notes |
|---------|-----------|-------|
| `[instructions]` | `.github/instructions/<name>.instructions.md` | Single file |
| `[agents]` | `.github/agents/<name>.agent.md` | Single file |
| `[prompts]` | `.github/prompts/<name>.prompt.md` | Single file |
| `[skills]` | `.github/skills/<name>/` | Entire directory (recursive) |

> **Note:** Skills are the only asset type downloaded as a directory. `cops` uses the GitHub Trees API to recursively fetch all files under the referenced path.

---

### `.cops.lock`

The lock file is automatically generated and records the exact resolved state of each asset. **You should not edit this file manually.**

```json
{
  "version": 1,
  "entries": {
    "agents/reviewer": {
      "type": "agents",
      "name": "reviewer",
      "ref": "my-org/copilot-agents/personas/senior-reviewer.md@main",
      "resolved_sha": "a1b2c3d4e5f6...",
      "target_path": ".github/agents/reviewer.agent.md",
      "checksum": "sha256-hex...",
      "synced_at": "2026-02-17T10:30:00Z"
    }
  }
}
```

The lock file:
- Pins the exact commit SHA that was resolved at sync time
- Stores a SHA-256 checksum of the downloaded content
- Records the timestamp of the last sync
- Enables `cops check` to detect drift

> **Recommendation:** Add `.cops.lock` to `.gitignore` if each developer should resolve independently, or commit it if you want fully reproducible environments across the team.

---

## 🔑 Authentication

`cops` uses a GitHub token for API access. It checks environment variables in this order:

1. `GITHUB_TOKEN`
2. `GH_TOKEN`

```bash
export GITHUB_TOKEN="ghp_your_token_here"
cops sync
```

### Public Repositories

If no token is set, `cops` falls back to **unauthenticated requests** with a warning. This works for public repositories but is subject to GitHub's stricter rate limits (60 requests/hour).

### Private Repositories

A token with `repo` scope is **required** to access private repositories.

> **Tip:** If you use the [GitHub CLI](https://cli.github.com/), `GH_TOKEN` is often already set in your environment.

---

## 🔄 CI/CD Integration

Use `cops check --strict` in your CI pipeline to ensure all Copilot assets are synced before merging.

### GitHub Actions

```yaml
name: Copilot Assets Check

on: [pull_request]

jobs:
  check-copilot-assets:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Install cops
        run: go install github.com/cbout22/copilot-sync/cmd/cops@latest

      - name: Sync assets
        run: cops sync
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Check assets are in sync
        run: cops check --strict
```

---

## 🧰 How It Works

```
copilot.toml
┌──────────────┐
│ [agents]     │
│ [instructions│   ┌─────────┐
│ [prompts]    │──▶│ GitHub  │
│ [skills]     │   │ API     │
└──────────────┘   └────┬────┘
                        │
                        ▼
                  .github/
                  ├── agents/
                  ├── instructions/
                  ├── prompts/
                  └── skills/
```

1. **Manifest** — `cops` reads `copilot.toml` to discover all declared assets
2. **Authentication** — Loads `GITHUB_TOKEN` / `GH_TOKEN` for GitHub API access
3. **Resolution** — For each entry, resolves `@latest` to the repo's default branch, builds the raw content URL
4. **Download** — Fetches file content (or recursively lists and downloads directory contents for skills)
5. **Injection** — Writes files to `.github/<type>/<name><extension>`

---

## 🤝 Contributing

WIP

## Roadmap

- Auto completion from github
- Enable other git repo
---
