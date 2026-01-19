# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-01-19

### Added

- `config` command to edit configuration in `$EDITOR`
- Default config template created when file doesn't exist
- Config validation on save with retry prompt
- Parallel cloning with worker pool (default: 4 workers)
- `-j/--jobs` flag to control parallel clone workers
- Progress spinner with animated display during clone operations
- TTY detection for graceful fallback in non-interactive environments

### Changed

- Cloning now runs in parallel for faster sync operations
- Improved output with progress tracking

## [0.1.0] - 2026-01-18

### Added

- Initial release
- `sync` command with manual strategy support
- Interactive diff display (green=new, gray=unchanged, red=orphaned)
- `-p/--prune` flag to delete orphaned repos
- `-a/--add` flag to add orphaned repos to config
- `--force` flag to skip confirmations
- `-c/--config` flag for custom config path
- Config file support (`$XDG_CONFIG_HOME/autogitter/config.yaml`)
- SSH key support per source
- Custom branch support per source
- MkDocs documentation with Material theme
- GitHub Actions for build, release, and docs deployment

[Unreleased]: https://github.com/arch-err/autogitter/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/arch-err/autogitter/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/arch-err/autogitter/releases/tag/v0.1.0
