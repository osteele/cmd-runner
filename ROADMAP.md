# cmd-runner Roadmap

This document outlines potential future enhancements for cmd-runner by canibalizing my unpublished package-script-runner. Since I abandoned that project when it became too complicated, I don't want to unreservedly port everything listed below into this new, smaller, project.

## Phase 1: Script Type Detection System ⭐

Add a script categorization system to better understand and organize commands:

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

## Phase 2: Enhanced Command Discovery

### Smart Pattern Matching
- Detect common script patterns across different tools
- Support for tool-specific conventions (e.g., "build:prod" vs "build-prod")
- Context-aware command selection based on project type

### Custom Script Parsing
- Parse pyproject.toml [tool.scripts] sections
- Parse Makefile includes and recursive makefiles
- Support for Taskfile.yml (Task runner)
- Parse Gradle/Maven custom tasks

## Phase 3: Additional Command Synonyms

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

## Phase 4: Additional Package Managers

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

## Phase 5: Interactive Features

### Command Selection UI
- List available commands with descriptions
- Group commands by type/phase
- Show keyboard shortcuts
- Search/filter commands

### Command History
- Track frequently used commands
- Suggest recent commands
- Per-project command history

## Phase 6: Configuration

### Project Configuration
- `.cmdrunner.yml` or `.cmdrunner.toml` for project-specific settings
- Custom command aliases
- Command chains/sequences
- Environment-specific commands

### Global Configuration
- User preferences (~/.config/cmdrunner/)
- Custom command templates
- Default flags for commands

## Phase 7: Advanced Features

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

## Phase 8: Developer Experience

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

## Phase 9: Performance Optimizations

### Caching
- Cache command discovery results
- Invalidate cache on file changes
- Persistent cache across sessions

### Lazy Loading
- Load runners on-demand
- Optimize startup time
- Parallel runner detection

## Phase 10: Integrations

### CI/CD Integration
- GitHub Actions support
- GitLab CI support
- Export commands as CI configuration

### IDE Integration
- VS Code extension
- IntelliJ plugin
- Task provider APIs

## Notes

Priority should be given to features that:
1. Improve command discovery accuracy
2. Reduce user friction
3. Support more development stacks
4. Enhance developer productivity

Features marked with ⭐ are recommended as high-priority items based on their impact on user experience.
