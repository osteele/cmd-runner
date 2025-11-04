# Project Specification

This document covers the internal design, architecture, and behavior of `cmd-runner`.

## Core Architecture

`cmd-runner` is built around a few key abstractions:

- **`CommandRunner`**: The main struct that manages the execution context, including the current directory and the project root. It orchestrates command discovery and execution.
- **`CommandSource` Interface**: An interface that each build system (like npm, cargo, or make) implements. It has methods to list available commands and find a specific command. This makes the tool extensible.

## Command Discovery and Execution

### Search Strategy

`cmd-runner` uses a multi-step process to find the right command to execute:

1.  First searches in the current directory for a matching command source (e.g., a `package.json` or `Makefile`).
2.  Then searches in the project root (determined by the presence of a `.jj` or `.git` directory).
3.  Tries command aliases (e.g., `fmt` for `format`, `dev` for `run`).

### Build System Priority

The tool searches for commands from different build systems in the following order of priority:

1.  **mise** - `.mise.toml` (polyglot runtime manager)
2.  **just** - `justfile` or `Justfile` (command runner)
3.  **make** - `Makefile` or `makefile` (classic build tool)
4.  **Node.js** - `package.json` with bun/pnpm/yarn/npm
5.  **Rust** - `Cargo.toml` (cargo)
6.  **Go** - `go.mod` (go modules)
7.  **Python** - `pyproject.toml` with uv
8.  **Java/Kotlin** - `build.gradle[.kts]` (gradle) or `pom.xml` (maven)

## Supported Commands and Aliases

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
| `setup` | - | - | Install dependencies for local development |
| `install` | - | - | Install binary/package globally |

**Note**: If a project has an actual command named `f`, `t`, etc., it will take precedence over the short alias expansion.

## Smart Command Synthesis

- **`check`**: If no native `check` command exists, `cmd-runner` automatically runs `lint`, `typecheck`, and `test` in sequence.
- **`fix`**: If no native `fix` command exists, it automatically runs `format` and `lint --fix`.
- **`typecheck`**: It will error if the project doesn't support type checking (e.g., no TypeScript, Python with pyright/mypy, Rust, or Go).

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
