# cmd-runner Roadmap

This document outlines potential future enhancements for cmd-runner by cannibalizing my unpublished package-script-runner. Since I abandoned that project when it became too complicated, I don't want to unreservedly port everything listed below into this new, smaller, project.

Features marked with 📦 come from package-script-runner. Features marked with ⭐ are high-priority.

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

### TUI Mode (Terminal User Interface) ⭐📦
- **Interactive command selection** with arrow keys
- **Keyboard shortcuts** for common scripts (1-9, letters)
- **Search/filter** commands with `/` key
- **Visual script grouping** by phase (Development, Quality, Build, etc.)
- **Script descriptions** and metadata display
- **Theme support** (dark/light/no-color)
- **Emoji indicators** for script types (optional)

### Project Management 📦
- **Save project directories** for quick access
- **Switch between projects** without changing directories
- **Project aliases** (e.g., `cmdr --project frontend test`)
- **Recent projects** list
- **Multi-project commands** (run same command in multiple projects)

### Configuration System 📦
- **Global config** (~/.cmdrunner.toml or ~/.config/cmdrunner/config.toml)
- **Project config** (.cmdrunner.toml in project root)
- **Settings include**:
  - Theme preferences
  - Default flags
  - Custom aliases
  - Project bookmarks
  - Emoji display preferences

### Enhanced Script Discovery 📦
- **Script metadata** from package files
- **Custom script descriptions**
- **Script dependencies** and relationships
- **Hidden/internal script filtering**
- **Priority ordering** for common scripts

## Script Type Detection System ⭐📦

Add a script categorization system to better understand and organize commands (from package-script-runner):

### Script Types
- **Development**: Run, Generate, Migration
- **Quality**: Test, Lint, TypeCheck, Format, Audit
- **Build**: Clean, Build, BuildDev, BuildProd
- **Dependencies**: Install, Update, Lock
- **Release**: Version, Publish, Deploy, DeployStaging, DeployProd

### Implementation
- Add ScriptType enumeration
- Implement pattern-based type detection
- Group commands by phase (Development, Quality, Build, Dependencies, Release)
- Use script types to provide better command suggestions
- Display with emojis/icons in TUI mode

## Enhanced Command Discovery 📦

### Smart Pattern Matching
- Detect common script patterns across different tools
- Support for tool-specific conventions (e.g., "build:prod" vs "build-prod")
- Context-aware command selection based on project type

### Custom Script Parsing
- Parse pyproject.toml [tool.scripts] sections
- Parse Makefile includes and recursive makefiles
- Support for Taskfile.yml (Task runner)
- Parse Gradle/Maven custom tasks

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

### Smart Command Resolution
- "ci" → appropriate install command for CI environments
- "precommit" → run lint, format, and test
- "all" → run common quality checks

## Additional Package Managers

### Ruby
- Bundler support (Gemfile)
- Rake task detection

### PHP
- Composer support (composer.json)
- Artisan command detection (Laravel)

### Elixir
- Mix support (mix.exs)

### .NET
- dotnet CLI support
- MSBuild detection

## Command History and Intelligence

### Command History
- Track frequently used commands
- Suggest recent commands
- Per-project command history
- Smart command suggestions based on context

## Additional Configuration Features

### Project Configuration
- `.cmdrunner.yml` or `.cmdrunner.toml` for project-specific settings
- Custom command aliases
- Command chains/sequences
- Environment-specific commands

### Global Configuration
- User preferences (~/.config/cmdrunner/)
- Custom command templates
- Default flags for commands

## Advanced Features

### Parallel Execution
- Run multiple commands concurrently
- Command dependencies and ordering
- Output aggregation

### Command Composition
- Chain commands with operators (&&, ||, ;)
- Pipe output between commands
- Conditional execution

### Environment Management
- Automatic environment variable loading (.env files)
- Environment-specific command variants
- Virtual environment activation

## Developer Experience

### Better Error Messages
- Suggest similar commands on typos
- Explain why a command wasn't found
- Provide setup instructions for missing tools

### Documentation Generation
- Auto-generate command documentation
- Export available commands as markdown
- Integration with project README

### Shell Completions
- Bash completion support
- Zsh completion support
- Fish completion support
- PowerShell completion support

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
- **Cross-platform** terminal compatibility

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
