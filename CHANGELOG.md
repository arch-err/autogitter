# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/arch-err/autogitter/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/arch-err/autogitter/releases/tag/v0.1.0
