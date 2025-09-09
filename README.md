# cmd-runner

[![CI](https://github.com/osteele/cmd-runner/actions/workflows/ci.yml/badge.svg)](https://github.com/osteele/cmd-runner/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/osteele/cmd-runner)](https://goreportcard.com/report/github.com/osteele/cmd-runner)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

<p align="center">
  <img src="docs/mascot.jpg" alt="cmd-runner mascot - a friendly robot with tools" width="256" height="256">
</p>

A smart command runner that finds and executes commands across different build systems and project types.

**Why?** Working across multiple projects with different tech stacks (Rust/Cargo, Node/npm/bun, Python/uv, Go, etc.) means remembering different commands for the same tasks. Instead of trying to recall whether it's `npm run test`, `cargo test`, `go test`, or `make test`, or wondering whether this Node project uses `npm run test` or `bun test`, simply use `cmdr test` in any project.

## Example

```bash
# In a Rust project
$ cmdr test
Running: cargo test

# In a Node project with bun
$ cmdr test
Running: bun run test

# In a project with a Makefile
$ cmdr format
Running: make format

# Or use the shorter alias
$ cr test
Running: cargo test

# Ultra-short aliases work too
$ cmdr t        # Same as: cmdr test
$ cmdr f        # Same as: cmdr format
$ cmdr b        # Same as: cmdr build
```

## Installation

```bash
go install github.com/osteele/cmd-runner/cmd/cmdr@latest
```

This installs the binary as `cmdr`. For an even shorter command, you can install the `cr` alias:

```bash
# Preview what will be changed (dry-run)
cmdr install-alias --dry-run

# Install the alias
cmdr install-alias
```

Or manually create an alias in your shell configuration:

```bash
alias cr=cmdr

# For an ultra-short alias, you could even use punctuation:
alias ,=cmdr   # Now you can type: , test
alias @=cmdr   # Now you can type: @ build
```

Or install from source:

```bash
git clone https://github.com/osteele/cmd-runner.git
cd cmd-runner
go install ./cmd/cmdr
```

## Usage

```bash
cmdr [OPTIONS] <command> [args...]
cmdr --list                      # List all available commands for current project
cmdr --help                      # Show help information
cmdr --version                   # Show version
cmdr install-alias [--dry-run]  # Install 'cr' alias to shell config
```

Options:
- `--list`, `-l` - List all available commands for current project
- `--version`, `-v` - Show version information
- `--help`, `-h` - Show help message

Examples:
```bash
cmdr format           # Runs format or fmt command
cmdr test             # Runs test command
cmdr run              # Runs run, dev, or serve command
cmdr build -- --prod  # Runs build with additional arguments

# Or with the short alias (after installing)
cr test               # Same as cmdr test
cr format             # Same as cmdr format

# Ultra-short command aliases
cmdr f                # format
cmdr t                # test
cmdr tc               # typecheck
cmdr r                # run
cmdr s                # serve/server
cmdr b                # build
cmdr l                # lint

# Show all available commands
cmdr --list           # List commands for current project
cmdr -l               # Short form of --list

# Flags are passed through to commands
cmdr test --verbose   # Runs test command with --verbose flag
cmdr build --prod     # Runs build command with --prod flag
```

### Discovering Available Commands

Use `cmdr --list` to see all available commands for your current project:

```bash
$ cmdr --list
Available commands for this project:

Node.js commands:
  lint         → npm run lint
  format       → npm run format
  dev          → npm run dev
  build        → npm run build
  test         → npm run test

Command aliases:
  f  → format     t  → test       tc → typecheck
  r  → run        s  → serve      b  → build
  l  → lint
```

The list command shows:
- Commands from your build system (make, just, npm scripts, etc.)
- What each command will actually execute
- Available short aliases
- Synthesized commands provided by cmd-runner

## Supported Commands

| Command | Aliases | Short | Description |
|---------|---------|-------|-------------|
| `format` | `fmt` | `f` | Code formatting |
| `test` | `tests` | `t` | Run tests |
| `typecheck` | `type-check`, `types` | `tc` | Run type checker |
| `run` | `dev`, `serve`, `start` | `r` | Run development server or application |
| `serve` | `dev`, `run`, `start` | `s` | Run server (alias for run) |
| `build` | - | `b` | Build the project |
| `lint` | - | `l` | Run linters |
| `check` | - | - | Run lint, typecheck, and test together |
| `fix` | - | - | Auto-fix issues |
| `clean` | - | - | Clean build artifacts |
| `install` | `setup` | - | Install dependencies |

**Note**: If a project has an actual command named `f`, `t`, etc., it will take precedence over the short alias expansion.

### Smart Command Synthesis

- **check**: If no native `check` command exists, automatically runs `lint`, `typecheck`, and `test` in sequence
- **fix**: If no native `fix` command exists, automatically runs `format` and `lint --fix`
- **typecheck**: Errors if the project doesn't support type checking (no TypeScript, Python with pyright/mypy, Rust, or Go)

## Supported Languages & Stacks

### JavaScript/TypeScript
- **Package Managers**: bun, pnpm, yarn, npm, deno
- **Type Checking**: TypeScript (`tsc`)
- **Common Tools**: biome, eslint, prettier

### Python
- **Package Manager**: uv (with pyproject.toml)
- **Type Checking**: pyright, mypy
- **Common Tools**: ruff, pytest

### Rust
- **Build System**: cargo
- **Type Checking**: Built-in (`cargo check`)
- **Common Tools**: clippy, rustfmt

### Go
- **Build System**: go modules
- **Type Checking**: Built-in (`go build`)
- **Common Tools**: go vet, gofmt

### Java/Kotlin
- **Build Systems**: gradle, maven
- **Type Checking**: Built-in compilation

## Supported Build Systems

The tool searches for commands in the following order:

1. **mise** - `.mise.toml` (polyglot runtime manager)
2. **just** - `justfile` or `Justfile` (command runner)
3. **make** - `Makefile` or `makefile` (classic build tool)
4. **Node.js** - `package.json` with bun/pnpm/yarn/npm
5. **Rust** - `Cargo.toml` (cargo)
6. **Go** - `go.mod` (go modules)
7. **Python** - `pyproject.toml` with uv
8. **Java/Kotlin** - `build.gradle[.kts]` (gradle) or `pom.xml` (maven)

## Search Strategy

1. First searches in the current directory
2. Then searches in the project root (determined by `.jj` or `.git` directory)
3. Tries command aliases (e.g., `fmt` for `format`, `dev` for `run`)

## Features

- Intelligent command aliasing (e.g., `run` → `dev` → `serve`)
- Searches both current directory and project root
- Passes through additional arguments to the underlying command
- Supports all major build systems and package managers

## See Also

Check out my [development tools](https://osteele.com/software/development-tools) page for additional projects.

## License

MIT License
