# cmd-runner Roadmap

This document outlines potential future enhancements for cmd-runner by cannibalizing my unpublished package-script-runner. Since I abandoned that project when it became too complicated, I don't want to unreservedly port everything listed below into this new, smaller, project.

Features marked with 📦 come from package-script-runner. Features marked with ⭐ are high-priority.

For more experimental and creative ideas, see [docs/ideas.txt](docs/ideas.txt).

## Recent Completions (January 2025)

### Architecture & Code Quality
- ✅ **Fixed string-matching bugs in command discovery** - Replaced unsafe `strings.Contains()` with exact map lookups
- ✅ **Implemented command discovery caching** - Added thread-safe cache to eliminate redundant shell-outs
- ✅ **Added priority-based source ordering** - CommandSources now properly sorted by Priority() values
- ✅ **Fixed resource leaks in MakeSource** - Replaced defer-in-loop patterns with `os.ReadFile()`
- ✅ **Removed dead code** - Deleted 124 lines of unused `findCommandExact` functions
- ✅ **Consolidated runner abstractions** - Removed duplicate `*runner` types in favor of CommandSource interface (saved 551 lines)
- ✅ **Fixed typecheck synthesis bugs** - Python typecheck now properly executes via package managers or directly
- ⬜ **Consolidate TypeScript typecheck synthesis** - Currently duplicated in sources_node.go and typecheck.go. Options: (A) centralize in typecheck.go for consistency with Python/Rust/Go, (B) extract shared helper function, or (C) keep Node sources "smart" about typecheck. Recommended: Option A or B for better maintainability.

## Command Execution Architecture ⭐

### Separate Planning and Execution Phases
- **Planning Phase**: Gather all commands to be executed, resolve aliases, detect capabilities
- **Execution Phase**: Execute the deduplicated plan with proper error handling
- **Benefits**:
  - Cleaner deduplication logic (replace current format/fmt hardcoded logic)
  - Better error reporting (show full plan before execution)
  - Easier to test and maintain
  - Extensible for future features (dry-run, confirmation prompts)

### Implementation
- Create CommandPlan struct with list of commands to execute
- Build plan based on command type and available runners
- Deduplicate equivalent commands (format/fmt, check/verify, etc.)
- Execute plan with progress reporting

## Features from package-script-runner 📦

### TUI Mode (Terminal User Interface) ⭐📦 ✅ IMPLEMENTED (2025-01-15)
- ✅ **Interactive command selection** with single-key shortcuts
- ✅ **Keyboard shortcuts** for common scripts (t, b, r, f, l, c, x, s)
- ✅ **Number keys** for additional commands (1-9)
- ✅ **Type-ahead** support with partial name matching
- ✅ **Toggle view** between menu and last output with `/` key
- ✅ **Repeat last command** with `.` key
- ✅ **Automatic flow** - success returns to menu, failures pause
- ✅ **Graceful exit** with `q` or Ctrl+C
- ⬜ **Visual script grouping** by phase (future enhancement)
- ⬜ **Script descriptions** and metadata display (partial - shows descriptions)
- ⬜ **Theme support** (future enhancement)
- ⬜ **Emoji indicators** for script types (future enhancement)

### Project Management 📦
- **Save project directories** for quick access
- **Switch between projects** without changing directories
- **Project aliases** (e.g., `cmdr --project frontend test`)
- **Recent projects** list
- **Multi-project commands** (run same command in multiple projects)

#

## Watch Mode Support ⭐

### Automatic Watch Mode Detection
- Detect if underlying commands support watch mode flags (--watch, -w)
- Check for dedicated watch scripts in package.json/pyproject.toml
- Support tool-specific watch patterns:
  - Node: `npm run test -- --watch`, `jest --watch`
  - Rust: `cargo watch -x test`
  - Go: `gow test`, `reflex`, `air`
  - Python: `pytest-watch`, `watchdog`
  - Make: `watchexec make test`

### Watch Mode Implementation
- Add --watch flag to cmdr commands: `cmdr test --watch`, `cmdr fix --watch`
- Fallback to generic file watchers (watchexec, entr) when native watch unavailable
- Smart file pattern detection based on project type
- Debouncing and intelligent re-run strategies

### Supported Commands in Watch Mode
- test → re-run tests on file changes
- fix → auto-fix on save
- format → auto-format on save
- lint → continuous linting
- typecheck → continuous type checking
- build → rebuild on changes
- run/dev → hot reload (if not already watching)

## Additional Command Synonyms

### Extended Aliases
- "i" → "install"
- "t" → "test"
- "b" → "build"
- "d" → "dev"
- "s" → "start"
- "f" → "format"
- "l" → "lint"
- "w" → "watch"

## Additional Package Managers

### Ruby
- Bundler support (Gemfile)
- Rake task detection

### PHP
- Composer support (composer.json)
- Artisan command detection (Laravel)

### Elixir
- Mix support (mix.exs)




## Developer Experience

### Better Error Messages
- Suggest similar commands on typos
- Explain why a command wasn't found
- Provide setup instructions for missing tools

### Python Type Checking
- Improve pyright/mypy fallback so synthesized `typecheck` uses `poetry run`/`uv run`

### Documentation Generation
- Auto-generate command documentation
- Export available commands as markdown
- Integration with project README

### Shell Completions
- Bash completion support
- Zsh completion support
- Fish completion support

## Performance Optimizations

### Caching
- Cache command discovery results
- Invalidate cache on file changes
- Persistent cache across sessions

### Lazy Loading
- Load runners on-demand
- Optimize startup time
- Parallel runner detection

## Integrations

### CI/CD Integration
- GitHub Actions support
- GitLab CI support
- Export commands as CI configuration

### IDE Integration
- VS Code extension
- IntelliJ plugin
- Task provider APIs

## Additional package-script-runner Features 📦

These features from package-script-runner could be considered for future phases:

### CLI Mode Features
- **Simple CLI interface** (non-TUI) with numbered/lettered shortcuts
- **Direct script execution** without UI (`cmdr test` runs test immediately)
- **List mode** (`--list` flag) to show available commands
- **Verbose output** mode for debugging

### Script Execution Features
- **Environment variable injection** for scripts
- **Working directory management** per script
- **Script output capture** and formatting
- **Error handling** with helpful messages

### UI/UX Enhancements
- **NO_COLOR** environment variable support
- **Accessibility features** for screen readers
- **Terminal detection** for appropriate UI selection

## Implementation Priority

### High Priority ⭐
1. Command Execution Architecture (Phase 0)
2. TUI Mode from package-script-runner
3. Script Type Detection System
4. Watch Mode Support

### Medium Priority
1. Configuration System
2. Project Management
3. Enhanced Command Discovery
4. Additional Command Synonyms

### Low Priority
1. Additional Package Managers
2. Advanced Features (parallel execution, etc.)
3. Performance Optimizations
4. IDE Integrations

## Notes

Priority should be given to features that:
1. Improve command discovery accuracy
2. Reduce user friction (especially features from package-script-runner that proved useful)
3. Support more development stacks
4. Enhance developer productivity

Features marked with ⭐ are recommended as high-priority items based on their impact on user experience.
Features marked with 📦 come from the package-script-runner project and have been validated through use.
