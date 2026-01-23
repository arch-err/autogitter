# Configuration

The configuration file is stored at `$XDG_CONFIG_HOME/autogitter/config.yaml` (typically `~/.config/autogitter/config.yaml`).

## Modular Configuration with sources.d

For better organization, you can split your sources across multiple files in the `sources.d` directory:

```
~/.config/autogitter/
├── config.yaml          # Main config (required)
└── sources.d/           # Additional source files (optional)
    ├── github.yaml
    ├── work.yaml
    └── personal.yaml
```

Each file in `sources.d/` uses the same format as the main config:

```yaml
# ~/.config/autogitter/sources.d/work.yaml
sources:
  - name: "Work Bitbucket"
    source: bitbucket.company.com/~username
    strategy: all
    type: bitbucket
    local_path: "~/Git/work"
```

**Rules:**
- Main `config.yaml` must exist
- Files are loaded in alphabetical order
- Only `.yaml` and `.yml` files are processed
- Sources from all files are merged together
- Not supported for remote configs (HTTP/SSH)

## Config Format

```yaml
sources:
  # Manual strategy - explicitly list repos
  - name: "GitHub (Personal)"
    source: github.com/username
    strategy: manual
    local_path: "~/Git/github"
    repos:
      - username/repo1
      - username/repo2

  # All strategy - sync all repos from user/org
  - name: "GitHub (All)"
    source: github.com/username
    strategy: all
    local_path: "~/Git/github-all"

  # Regex strategy - filter repos by pattern
  - name: "APIs Only"
    source: github.com/myorg
    strategy: regex
    local_path: "~/Git/apis"
    regex_strategy:
      pattern: "^myorg/api-.*"

  # Gitea with explicit type
  - name: "Work Gitea"
    source: gitea.company.com/myuser
    strategy: all
    type: gitea
    local_path: "/work/git"

  # Bitbucket Server with SSH options
  - name: "Bitbucket"
    source: bitbucket.company.com/~username
    strategy: all
    type: bitbucket
    local_path: "~/Git/bitbucket"
    ssh_options:
      port: 7999
      private_key: "~/.ssh/work_ed25519"
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name for the source |
| `source` | Yes | Git host and user/org (e.g., `github.com/username`) |
| `strategy` | Yes | Sync strategy: `manual`, `all`, `regex`, or `file` |
| `type` | No | Provider type: `github`, `gitea`, `bitbucket` (auto-detected from host if omitted) |
| `local_path` | Yes | Where to clone repos (supports `$HOME`, `~`) |
| `repos` | For manual | List of repos to sync |
| `regex_strategy` | For regex | Regex pattern configuration |
| `branch` | No | Branch to clone (uses remote default if not set) |
| `private_key` | No | Path to SSH key for this source (legacy, prefer `ssh_options`) |
| `ssh_options` | No | SSH configuration (port, private key) |

## SSH Options

For sources that require custom SSH settings (like Bitbucket Server with non-standard ports):

```yaml
ssh_options:
  port: 7999                          # Custom SSH port
  private_key: "~/.ssh/work_ed25519"  # Path to SSH private key
```

When `ssh_options.port` is specified, autogitter uses the `ssh://` URL format:
```
ssh://git@host:port/repo.git
```

## Strategies

### Manual

Explicitly list repositories to sync:

```yaml
strategy: manual
repos:
  - user/repo1
  - user/repo2
  - org/project
```

Best for: Curated lists of specific repos you want to track.

### All

Sync all repositories from a user/organization. Requires API authentication.

```yaml
strategy: all
```

This fetches all non-archived repositories from the specified user or organization.

Best for: Backing up all your repos or keeping a local mirror.

### Regex

Sync repositories matching a regex pattern. Requires API authentication.

```yaml
strategy: regex
regex_strategy:
  pattern: "^myorg/api-.*"  # matches repos starting with "api-"
```

The pattern is matched against the full repository name (e.g., `username/repo-name`).

Examples:
- `^user/.*` - All repos from user
- `.*-service$` - Repos ending with "-service"
- `^org/(api|web)-.*` - Repos starting with "api-" or "web-"

Best for: Syncing a subset of repos based on naming conventions.

### File (Coming Soon)

Sync repositories containing a specific file:

```yaml
strategy: file
file_strategy:
  filename: ".autogitter"
```

## Provider Types

Autogitter auto-detects the provider from the host:

| Host | Detected Type |
|------|---------------|
| `github.com` | `github` |
| `bitbucket.org` | `bitbucket` |
| Other | `gitea` (default) |

For self-hosted instances, specify `type` explicitly:

```yaml
- name: "Self-hosted Bitbucket"
  source: scm.company.com/~username
  type: bitbucket  # Required for self-hosted
  strategy: all
  local_path: "~/Git/work"
```

## Authentication

The `all` and `regex` strategies require API tokens. Set up authentication with:

```bash
ag connect
```

Tokens are stored in `$XDG_DATA_HOME/autogitter/credentials.env`.

### Environment Variables

| Provider | Environment Variable |
|----------|---------------------|
| GitHub | `GITHUB_TOKEN` |
| Gitea | `GITEA_TOKEN` |
| Bitbucket | `BITBUCKET_TOKEN` |

You can also export these directly:

```bash
export GITHUB_TOKEN=ghp_xxxx
ag sync
```

## Remote Configs

Load configuration from remote sources using the `-c` flag:

### HTTP/HTTPS

```bash
ag sync -c https://example.com/config.yaml
ag config -v -c https://raw.githubusercontent.com/user/repo/main/config.yaml
```

### SSH

```bash
ag sync -c user@host:/path/to/config.yaml
ag sync -c ssh://user@host/path/to/config.yaml
```

Remote configs can be used with `sync` and `config --validate`, but cannot be edited.

## Environment Variable Expansion

Paths support environment variable expansion:

- `$HOME` or `~` - User's home directory
- `$XDG_CONFIG_HOME` - XDG config directory
- Any other environment variable

```yaml
local_path: "$HOME/Git/repos"
local_path: "~/Git/repos"
local_path: "/data/$USER/repos"
```

## Custom Config Path

Use a custom config file:

```bash
ag sync -c /path/to/config.yaml
ag sync -c ~/dotfiles/autogitter.yaml
```
