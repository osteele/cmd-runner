package cmdrunner

import (
	"fmt"
	"os"
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
		if cmd := r.FindCommand(dir); cmd != nil {
			return r.ExecuteCommand(cmd)
		}
	}

	// If no native fix command, synthesize by running format and lint fix commands
	return r.synthesizeFixCommand()
}

// synthesizeFixCommand runs format and lint fix commands
func (r *CommandRunner) synthesizeFixCommand() error {
	// Commands to try for fixing, in order of preference
	fixCommands := []struct {
		command string
		args    []string
	}{
		{"format", nil},
		{"fmt", nil},
		{"lint", []string{"--fix"}},
	}

	var foundAny bool
	var executedCommands []string
	var hasErrors bool

	// First check if any fix-related commands are available
	for _, fc := range fixCommands {
		tempRunner := &CommandRunner{
			Command:     fc.command,
			Args:        fc.args,
			CurrentDir:  r.CurrentDir,
			ProjectRoot: r.ProjectRoot,
		}
		if tempRunner.hasCommand(fc.command) {
			foundAny = true
		}
	}

	if !foundAny {
		return fmt.Errorf("no fix, format, or lint commands found")
	}

	fmt.Fprintf(os.Stderr, "Running fix (synthesizing from available commands)...\n")

	for _, fc := range fixCommands {
		tempRunner := &CommandRunner{
			Command:     fc.command,
			Args:        append(fc.args, r.Args...),
			CurrentDir:  r.CurrentDir,
			ProjectRoot: r.ProjectRoot,
		}

		if !tempRunner.hasCommand(fc.command) {
			continue
		}

		cmdDisplay := fc.command
		if len(fc.args) > 0 {
			cmdDisplay = fmt.Sprintf("%s %s", fc.command, strings.Join(fc.args, " "))
		}

		fmt.Fprintf(os.Stderr, "\n→ Running %s...\n", cmdDisplay)

		if err := tempRunner.Run(); err != nil {
			// For fix commands, we often want to continue even if one fails
			hasErrors = true
			fmt.Fprintf(os.Stderr, "  ✗ %s failed: %v\n", cmdDisplay, err)
		} else {
			executedCommands = append(executedCommands, cmdDisplay)
		}
	}

	if len(executedCommands) == 0 && hasErrors {
		return fmt.Errorf("fix failed: no commands succeeded")
	}

	return nil
}
