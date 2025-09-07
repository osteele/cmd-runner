package cmdrunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HandleFixCommand handles the special 'fix' command that runs format/lint fixes
func HandleFixCommand(r *CommandRunner) error {
	dirs := []string{r.CurrentDir}
	if r.ProjectRoot != r.CurrentDir {
		dirs = append(dirs, r.ProjectRoot)
	}

	// Try to find a native fix command first
	for _, dir := range dirs {
		if cmd := r.findNativeFixCommand(dir); cmd != nil {
			return r.ExecuteCommand(cmd)
		}
	}

	// If no native fix command, synthesize by running format and lint fix commands
	return r.synthesizeFixCommand()
}

// findNativeFixCommand looks for a native fix command in the project
func (r *CommandRunner) findNativeFixCommand(dir string) *exec.Cmd {
	// Create a temporary runner to check for fix command
	tempRunner := &CommandRunner{
		Command:     "fix",
		Args:        r.Args,
		CurrentDir:  dir,
		ProjectRoot: r.ProjectRoot,
	}
	return tempRunner.FindCommand(dir)
}

// synthesizeFixCommand runs format and lint fix commands
func (r *CommandRunner) synthesizeFixCommand() error {
	// Commands to try for fixing, in order of preference
	// Note: We'll deduplicate format/fmt below
	fixCommands := []struct {
		command string
		args    []string
	}{
		{"format", nil},
		{"fmt", nil},
	}

	// Add lint --fix only for projects that support it
	if r.supportsLintFix() {
		fixCommands = append(fixCommands, struct {
			command string
			args    []string
		}{"lint", []string{"--fix"}})
	}

	var foundAny bool
	var executedCommands []string
	var hasErrors bool

	// First check if any fix-related commands are available
	for _, fc := range fixCommands {
		if r.hasCommand(fc.command) {
			foundAny = true
		}
	}

	if !foundAny {
		return fmt.Errorf("no fix, format, or lint commands found")
	}

	fmt.Fprintf(os.Stderr, "Running fix (synthesizing from available commands)...\n")

	// Track what we've already run to avoid duplicates
	executedTypes := make(map[string]bool)

	for _, fc := range fixCommands {
		if !r.hasCommand(fc.command) {
			continue
		}

		// Skip if we already ran a formatting command (format and fmt are equivalent)
		if (fc.command == "format" || fc.command == "fmt") && executedTypes["format"] {
			continue
		}

		cmdDisplay := fc.command
		if len(fc.args) > 0 {
			cmdDisplay = fmt.Sprintf("%s %s", fc.command, strings.Join(fc.args, " "))
		}

		fmt.Fprintf(os.Stderr, "\n→ Running %s...\n", cmdDisplay)

		tempRunner := &CommandRunner{
			Command:     fc.command,
			Args:        append(fc.args, r.Args...),
			CurrentDir:  r.CurrentDir,
			ProjectRoot: r.ProjectRoot,
		}

		if err := tempRunner.Run(); err != nil {
			// For fix commands, we often want to continue even if one fails
			hasErrors = true
			fmt.Fprintf(os.Stderr, "  ✗ %s failed: %v\n", cmdDisplay, err)
		} else {
			executedCommands = append(executedCommands, cmdDisplay)
			// Mark format as executed for both format and fmt commands
			if fc.command == "format" || fc.command == "fmt" {
				executedTypes["format"] = true
			}
		}
	}

	if len(executedCommands) == 0 && hasErrors {
		return fmt.Errorf("fix failed: no commands succeeded")
	}

	return nil
}

// supportsLintFix checks if the project's lint command supports a --fix flag
func (r *CommandRunner) supportsLintFix() bool {
	// Go projects don't support lint --fix (go vet has no --fix flag)
	if FileExists(filepath.Join(r.CurrentDir, "go.mod")) ||
		FileExists(filepath.Join(r.ProjectRoot, "go.mod")) {
		return false
	}

	dirs := []string{r.CurrentDir}
	if r.ProjectRoot != r.CurrentDir {
		dirs = append(dirs, r.ProjectRoot)
	}

	for _, dir := range dirs {
		// Node.js projects with ESLint typically support --fix
		if FileExists(filepath.Join(dir, "package.json")) {
			if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
				content := string(data)
				if strings.Contains(content, "eslint") {
					return true
				}
			}
		}

		// Python projects with ruff support --fix
		if FileExists(filepath.Join(dir, "pyproject.toml")) {
			if data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml")); err == nil {
				content := string(data)
				if strings.Contains(content, "ruff") {
					return true
				}
			}
		}

		// Rust clippy supports --fix
		if FileExists(filepath.Join(dir, "Cargo.toml")) {
			// For Rust, we'd actually want to run "cargo fix" or "cargo clippy --fix"
			// but for now return false since our lint command maps to "cargo clippy"
			// which doesn't take --fix as a trailing argument
			return false
		}
	}

	return false
}
