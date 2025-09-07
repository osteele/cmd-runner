# Development

## Prerequisites

- [mise](https://mise.jdx.dev/) - Tool version manager and task runner

Install mise:
```bash
# Using curl
curl https://mise.run | sh

# Or using homebrew on macOS
brew install mise
```

## Setup

Clone the repository and set up the development environment:

```bash
git clone https://github.com/osteele/cmd-runner.git
cd cmd-runner

# Install all development tools (Go, golangci-lint, lefthook)
mise install

# Install Go dependencies
go mod download

# Install git hooks
lefthook install

# Verify everything is working
mise run check
```

That's it! Mise will automatically install the correct versions of:
- Go 1.25
- golangci-lint 2.x
- Lefthook 1.x

No need to manually install Go or any other tools.

## Development Commands

```bash
# Format code
mise run format  # or mise run fmt

# Run linting
mise run lint

# Run tests
mise run test

# Generate coverage report
mise run coverage

# Build binary
mise run build

# Install locally
mise run install

# Clean artifacts
mise run clean

# Run all checks (format, lint, test)
mise run check

# Build for all platforms
mise run release
```

### Quick Reference

All tools are managed by mise and available in your PATH after `mise install`:

- `go` - Go compiler and tools
- `golangci-lint` - Comprehensive Go linter
- `lefthook` - Git hooks manager

Direct tool usage:

```bash
# These commands work directly after mise install
go fmt ./...
go test ./...
go build -o cmdr ./cmd/cmdr
golangci-lint run
lefthook run pre-push
```


## Git Hooks with Lefthook

This project uses [Lefthook](https://github.com/evilmartians/lefthook) for managing Git hooks. Lefthook is a fast, cross-platform Git hooks manager written in Go.

Lefthook is automatically installed when you run `mise install` during setup. The git hooks are installed with:

```bash
lefthook install
```

This sets up hooks that automatically run on commits and pushes.

### Manual Hook Runs

Run all hooks:

```bash
lefthook run pre-commit
lefthook run pre-push
```

Run specific hooks:

```bash
lefthook run pre-commit --commands go-fmt
lefthook run pre-push --commands go-test
```

### What Gets Checked

The Lefthook configuration (`lefthook.yml`) includes:

**Pre-commit hooks** (run in parallel):
- **go-fmt**: Checks Go code formatting
- **go-vet**: Runs static analysis
- **go-imports**: Checks import statements (if goimports is installed)

**Pre-push hooks** (run sequentially):
- **go-fmt-check**: Ensures all code is formatted
- **go-vet**: Runs comprehensive static analysis
- **go-test**: Runs all tests
- **go-build**: Verifies the project builds

**Commit-msg hook**:
- Validates commit message is not empty

### Skipping Hooks

To skip hooks temporarily:

```bash
# Skip all hooks
LEFTHOOK=0 git commit -m "message"

# Or use --no-verify
git commit --no-verify -m "message"
```

## Project Structure

```
cmd-runner/
├── cmd/
│   └── cmdr/
│       └── main.go      # CLI entry point
├── cmdrunner.go         # Main library package
├── runners.go           # Build system runners
├── check.go            # Check command implementation
├── cmdrunner_test.go   # Tests for cmdrunner package
├── runners_test.go     # Tests for runners
├── lefthook.yml        # Git hooks configuration
├── .mise.toml          # mise task configuration
├── go.mod              # Go module definition
├── go.sum              # Dependency checksums
├── README.md           # User documentation
├── DEVELOPMENT.md      # This file
├── LICENSE             # MIT License
└── .gitignore          # Git ignore rules
```

## Testing Strategy

- Unit tests for individual functions
- Integration tests for command detection
- Mock filesystem operations where needed
- Test coverage target: 80%+

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## Release Process

1. Update version in code if applicable
2. Run full test suite
3. Build binaries for all platforms
4. Create git tag: `git tag -a v1.0.0 -m "Release v1.0.0"`
5. Push tag: `git push origin v1.0.0`
6. Create GitHub release with binaries
