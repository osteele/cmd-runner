# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2025-12-11

### Fixed

- Improve terminal width detection

### Changed

- Improve alias detection and typecheck handling
- Update golangci-lint version and configuration

## [0.1.0] - 2025-11-04

Initial public release.

### Added

- Unified command runner supporting multiple build systems (npm, pnpm, yarn, bun, uv, poetry, cargo, go, make, mise, just)
- `--list` command to discover available commands for current project
- Interactive TUI mode (`--interactive`, `-i`) for command selection
- Ultra-short command aliases (`f` for format, `t` for test, `b` for build, etc.)
- Distinguish between `setup` (install dependencies) and `install` (install globally) commands
- Command caching and priority sorting
- Type checking command synthesis for TypeScript/Python projects
- Hide private commands from `--list` output
- Deno and Poetry runner support
- `install-alias` command to add `cr` alias to shell config

### Changed

- Renamed binary from `cmd-runner` to `cmdr`
- Reorganized internal package structure

[0.2.0]: https://github.com/osteele/cmd-runner/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/osteele/cmd-runner/releases/tag/v0.1.0
