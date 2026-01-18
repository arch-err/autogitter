# Configuration

The configuration file is stored at `$XDG_CONFIG_HOME/autogitter/config.yaml` (typically `~/.config/autogitter/config.yaml`).

## Config Format

```yaml
sources:
  - name: "Github (Personal)"
    source: github.com/username
    strategy: manual
    local_path: "$HOME/Git/github"
    repos:
      - username/repo1
      - username/repo2

  - name: "Work Gitea"
    source: gitea.company.com/myuser
    strategy: manual
    local_path: "/work/git"
    private_key: "~/.ssh/work_ed25519"
    branch: master
    repos:
      - myuser/project1
      - team/shared-project
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name for the source |
| `source` | Yes | Git host and user/org (e.g., `github.com/username`) |
| `strategy` | Yes | Sync strategy: `manual`, `all`, or `file` |
| `local_path` | Yes | Where to clone repos (supports `$HOME`, `~`) |
| `repos` | For manual | List of repos to sync |
| `private_key` | No | Path to SSH key for this source |
| `branch` | No | Default branch (defaults to `main`) |

## Strategies

### Manual

Explicitly list repositories to sync:

```yaml
strategy: manual
repos:
  - user/repo1
  - user/repo2
```

### All (Coming Soon)

Sync all repositories from a user/organization:

```yaml
strategy: all
```

### File (Coming Soon)

Sync repositories containing a specific file:

```yaml
strategy: file
file_strategy:
  filename: "ag"  # or your custom filename
```

## Environment Variables

Paths support environment variable expansion:

- `$HOME` or `~` - User's home directory
- `$XDG_CONFIG_HOME` - XDG config directory
- Any other environment variable

## Custom Config Path

Use the `-c` or `--config` flag to specify a custom config file:

```bash
ag sync -c /path/to/config.yaml
```
