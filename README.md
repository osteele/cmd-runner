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
cmdr <command> [args...]
cmdr install-alias [--dry-run]  # Install 'cr' alias to shell config
```

Examples:
```bash
cmdr format           # Runs format or fmt command
cmdr test             # Runs test command
cmdr run              # Runs run, dev, or serve command
cmdr build -- --prod  # Runs build with additional arguments

# Or with the short alias (after installing)
cr test               # Same as cmdr test
cr format             # Same as cmdr format
```

## Supported Commands

- `format` / `fmt` - Code formatting
- `run` / `dev` / `serve` / `start` - Run development server or application
- `build` - Build the project
- `lint` - Run linters
- `typecheck` - Run type checker (TypeScript, Python, Rust, Go)
- `test` - Run tests
- `check` - Run lint, typecheck, and test together
- `fix` - Auto-fix issues
- `clean` - Clean build artifacts
- `install` / `setup` - Install dependencies

## Supported Build Systems

The tool searches for commands in the following order:

1. **mise** - `.mise.toml`
2. **just** - `justfile` or `Justfile`
3. **make** - `Makefile` or `makefile`
4. **npm** - `package.json` (with npm)
5. **bun** - `package.json` (with bun.lockb)
6. **cargo** - `Cargo.toml` (Rust)
7. **go** - `go.mod`
8. **uv** - `pyproject.toml` (Python with uv)
9. **gradle** - `build.gradle` or `build.gradle.kts`
10. **maven** - `pom.xml`

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
