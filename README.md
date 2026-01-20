<p align="center">
  <!-- Logo placeholder - replace with your logo -->
  <img src="docs/assets/logo.png" alt="Autogitter Logo" width="200">
</p>

<h1 align="center">Autogitter</h1>

<p align="center">
  <strong>Git repository synchronization tool for managing multiple repos across providers</strong>
</p>

<p align="center">
  <a href="https://github.com/arch-err/autogitter/releases/latest"><img src="https://img.shields.io/github/v/release/arch-err/autogitter?style=flat-square&color=orange" alt="Release"></a>
  <a href="https://github.com/arch-err/autogitter/actions/workflows/build.yaml"><img src="https://img.shields.io/github/actions/workflow/status/arch-err/autogitter/build.yaml?style=flat-square" alt="Build"></a>
  <a href="https://github.com/arch-err/autogitter/blob/main/LICENSE"><img src="https://img.shields.io/github/license/arch-err/autogitter?style=flat-square" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/arch-err/autogitter"><img src="https://goreportcard.com/badge/github.com/arch-err/autogitter?style=flat-square" alt="Go Report"></a>
</p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go"></a>
  <a href="https://cobra.dev/"><img src="https://img.shields.io/badge/Cobra-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Cobra"></a>
  <a href="https://charm.sh/"><img src="https://img.shields.io/badge/Charm-FF69B4?style=flat-square&logo=data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI+PHBhdGggZmlsbD0id2hpdGUiIGQ9Ik0xMiAyQzYuNDggMiAyIDYuNDggMiAxMnM0LjQ4IDEwIDEwIDEwIDEwLTQuNDggMTAtMTBTMTcuNTIgMiAxMiAyem0wIDE4Yy00LjQyIDAtOC0zLjU4LTgtOHMzLjU4LTggOC04IDggMy41OCA4IDgtMy41OCA4LTggOHoiLz48L3N2Zz4=" alt="Charm"></a>
  <a href="https://claude.ai/"><img src="https://img.shields.io/badge/Built%20with-Claude%20Code-blueviolet?style=flat-square" alt="Built with Claude Code"></a>
</p>

<p align="center">
  <a href="https://arch-err.github.io/autogitter">Documentation</a> •
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#features">Features</a>
</p>

---

## Features

| Feature | Description |
|---------|-------------|
| **Multi-Provider** | GitHub, Gitea, and Bitbucket (Cloud + Server) |
| **Sync Strategies** | `manual` (explicit list), `all` (fetch from API), `regex` (pattern matching) |
| **Parallel Operations** | Clone and pull with configurable worker pools |
| **Remote Configs** | Load config from HTTP/HTTPS URLs or SSH paths |
| **Dry Run** | Preview changes before applying them |
| **Interactive CLI** | Beautiful diffs, prompts, and progress indicators |
| **SSH Options** | Custom ports and private keys per source |

## Installation

### From Releases (Recommended)

Download the latest binary from the [releases page](https://github.com/arch-err/autogitter/releases).

```bash
# Linux (amd64)
curl -L https://github.com/arch-err/autogitter/releases/latest/download/ag-linux-amd64 -o ag
chmod +x ag
sudo mv ag /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/arch-err/autogitter/releases/latest/download/ag-darwin-arm64 -o ag
chmod +x ag
sudo mv ag /usr/local/bin/

# Windows - download ag-windows-amd64.exe from releases
```

### From Source

```bash
go install github.com/arch-err/autogitter/cmd/ag@latest
```

## Quick Start

**1. Set up authentication (for `all`/`regex` strategies):**

```bash
ag connect
```

**2. Create your config:**

```bash
ag config
```

This opens your config in `$EDITOR`. Example configuration:

```yaml
sources:
  # Sync all repos from a GitHub user
  - name: "GitHub"
    source: github.com/your-username
    strategy: all
    local_path: "~/Git/github"

  # Sync specific repos manually
  - name: "Work"
    source: gitea.company.com/myteam
    strategy: manual
    local_path: "~/Git/work"
    repos:
      - myteam/project-alpha
      - myteam/project-beta

  # Sync repos matching a pattern
  - name: "APIs"
    source: github.com/myorg
    strategy: regex
    local_path: "~/Git/apis"
    regex_strategy:
      pattern: "^myorg/api-.*"
```

**3. Sync your repositories:**

```bash
ag sync
```

**4. Keep repos updated:**

```bash
ag pull
```

## Usage

```bash
# Sync repositories (clone new, detect orphaned)
ag sync

# Preview what would happen
ag sync --dry-run

# Sync with 8 parallel workers
ag sync -j 8

# Prune repos not in config
ag sync --prune

# Pull updates for all repos
ag pull

# Edit config
ag config

# Validate config
ag config --validate

# Generate config template to stdout
ag config --generate > config.yaml

# Use a remote config
ag sync -c https://example.com/config.yaml
ag sync -c user@host:/path/to/config.yaml
```

## Providers

| Provider | Host Detection | Token Env Var |
|----------|---------------|---------------|
| GitHub | `github.com` | `GITHUB_TOKEN` |
| Gitea | Custom hosts | `GITEA_TOKEN` |
| Bitbucket | `bitbucket.org` or custom | `BITBUCKET_TOKEN` |

For self-hosted instances, specify the `type` field explicitly:

```yaml
- name: "Self-hosted Gitea"
  source: git.company.com/user
  type: gitea  # explicit type
  strategy: all
  local_path: "~/Git/company"
```

## Documentation

Full documentation available [here](https://arch-err.github.io/autogitter).

## Built With

- [Go](https://go.dev/) - Programming language
- [Cobra](https://cobra.dev/) - CLI framework
- [Charm](https://charm.sh/) - Terminal UI libraries (Huh, Lip Gloss, Log)
- [Claude Code](https://claude.ai/code) - AI pair programming

## License

MIT License - see [LICENSE](LICENSE) for details.
