# Autogitter Development Context

## Project Overview

**Autogitter** (`ag`) is a Git repository synchronization tool written in Go. It helps manage multiple repositories across different providers (GitHub, Gitea, Bitbucket).

- **Repo**: https://github.com/arch-err/autogitter
- **Docs**: https://arch-err.github.io/autogitter
- **Current Version**: v0.7.0

## Tech Stack

- **Language**: Go 1.24
- **CLI Framework**: [Cobra](https://cobra.dev/)
- **Terminal UI**: [Charm](https://charm.sh/) (huh, lipgloss, log)
- **Docs**: MkDocs with [terminal theme](https://github.com/ntno/mkdocs-terminal) (gruvbox_dark palette)
- **CI/CD**: GitHub Actions (build, lint, release, docs)

## Project Structure

```
autogitter/
├── cmd/ag/main.go          # CLI entry point, all commands defined here
├── internal/
│   ├── config/             # Config loading, validation, templates
│   ├── connector/          # API connectors (GitHub, Gitea, Bitbucket)
│   ├── git/                # Git operations (clone, pull)
│   ├── sync/               # Sync logic, status computation
│   └── ui/                 # Terminal UI (diffs, prompts, clipboard)
├── docs/                   # MkDocs documentation
│   ├── index.md
│   ├── installation.md
│   ├── configuration.md
│   └── usage.md
├── .github/workflows/
│   ├── build.yaml          # Build + lint on push to main
│   ├── release.yaml        # Cross-compile releases on tag push
│   └── docs.yaml           # Deploy MkDocs to GitHub Pages
├── mkdocs.yml              # MkDocs configuration
├── go.mod / go.sum
├── Taskfile.yaml           # Task runner config
└── LICENSE                 # MIT
```

## Commands

| Command | Description |
|---------|-------------|
| `ag sync` | Clone missing repos, detect orphaned ones |
| `ag pull` | Pull updates for all local repos |
| `ag diff` | Show unified diff of local vs config state |
| `ag config` | Edit/validate config file |
| `ag connect` | Set up API authentication |

### Key Flags

- `--config, -c` - Custom config path (local, HTTP, or SSH)
- `--debug` - Enable debug logging
- `--dry-run, -n` - Preview changes (sync)
- `--jobs, -j` - Parallel workers (sync, pull)
- `--prune, -p` / `--add, -a` - Handle orphaned repos (sync)

## Config Location

- Config: `$XDG_CONFIG_HOME/autogitter/config.yaml` (~/.config/autogitter/config.yaml)
- Credentials: `$XDG_DATA_HOME/autogitter/credentials.env` (~/.local/share/autogitter/credentials.env)

## Sync Strategies

1. **manual** - Explicit list of repos
2. **all** - Fetch all repos from user/org via API
3. **regex** - Filter repos by pattern
4. **file** - (Coming soon) Repos containing specific file

## Recent Changes (v0.7.0)

- Added `ag diff` command with unified diff output
- Added `ag pull` command for batch updates
- Added `--dry-run` flag to sync
- Added clipboard support (copies token URL in `ag connect`)
- Added Bitbucket connector (Cloud + Server)
- Added remote config support (HTTP/SSH)
- Fixed CI/CD: X11 dependencies for clipboard package, Go 1.24

## CI/CD Notes

- **Build workflow** requires `libx11-dev` for clipboard package
- **Lint job** also needs X11 deps (golangci-lint compiles the code)
- **Release workflow** cross-compiles for linux/darwin/windows (amd64/arm64)
- Workflows ignore docs-only changes via `paths-ignore`

## Key Implementation Details

### Clipboard (internal/ui/ui.go)
Uses `golang.design/x/clipboard` with OSC 52 escape sequence fallback for terminal compatibility.

### Diff Output (internal/ui/ui.go)
`PrintUnifiedDiff()` renders colored diff-style output:
- `+` green = repo to clone
- `-` red = orphaned repo
- ` ` gray = unchanged

### Status Computation (internal/sync/sync.go)
`ComputeSourceStatus()` compares local directory against config/API to determine repo states.

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `GITHUB_TOKEN` | GitHub API auth |
| `GITEA_TOKEN` | Gitea API auth |
| `BITBUCKET_TOKEN` | Bitbucket API auth |
| `EDITOR` / `VISUAL` | Editor for `ag config` |

## Useful Commands

```bash
# Build
go build -o ag ./cmd/ag

# Run tests
go test -v ./...

# Lint
golangci-lint run

# Serve docs locally
mkdocs serve

# Tag a release
git tag v0.x.0 && git push --tags
```
