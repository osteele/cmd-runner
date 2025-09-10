package cmdrunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HandleTypecheckCommand handles the special 'typecheck' command
func HandleTypecheckCommand(r *CommandRunner) error {
	dirs := []string{r.CurrentDir}
	if r.ProjectRoot != r.CurrentDir {
		dirs = append(dirs, r.ProjectRoot)
	}

	// First try to find a native typecheck command
	for _, dir := range dirs {
		project := ResolveProject(dir)
		for _, source := range project.CommandSources {
			if cmd := source.FindCommand("typecheck", r.Args); cmd != nil {
				return r.ExecuteCommand(cmd)
			}
		}
	}

	// Check if this project type supports typechecking
	if !r.hasTypecheckCapability() {
		return fmt.Errorf("no typecheck command or type checking capability found for this project")
	}

	// Try to synthesize a typecheck command based on project type
	return r.synthesizeTypecheckCommand()
}

// synthesizeTypecheckCommand creates a typecheck command based on project type
func (r *CommandRunner) synthesizeTypecheckCommand() error {
	dirs := []string{r.CurrentDir}
	if r.ProjectRoot != r.CurrentDir {
		dirs = append(dirs, r.ProjectRoot)
	}

	for _, dir := range dirs {
		// TypeScript projects - use tsc
		if FileExists(filepath.Join(dir, "tsconfig.json")) {
			packageManager := detectPackageManager(dir)
			if packageManager != "" {
				fmt.Fprintf(os.Stderr, "Running typecheck using tsc...\n")
				cmd := r.createTypescriptCheckCommand(dir, packageManager)
				if cmd != nil {
					return r.ExecuteCommand(cmd)
				}
			}
		}

		// Python projects - try pyright or mypy
		if FileExists(filepath.Join(dir, "pyproject.toml")) {
			data, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
			content := string(data)

			var cmd *CommandRunner
			if strings.Contains(content, "pyright") {
				cmd = &CommandRunner{
					Command:     "pyright",
					Args:        r.Args,
					CurrentDir:  r.CurrentDir,
					ProjectRoot: r.ProjectRoot,
				}
			} else if strings.Contains(content, "mypy") {
				cmd = &CommandRunner{
					Command:     "mypy",
					Args:        append([]string{"."}, r.Args...),
					CurrentDir:  r.CurrentDir,
					ProjectRoot: r.ProjectRoot,
				}
			}

			if cmd != nil {
				fmt.Fprintf(os.Stderr, "Running typecheck using %s...\n", cmd.Command)
				return cmd.Run()
			}
		}

		// Rust projects - use cargo check
		if FileExists(filepath.Join(dir, "Cargo.toml")) {
			fmt.Fprintf(os.Stderr, "Running typecheck using cargo check...\n")
			// Find cargo runner and execute
			cargoRunner := &cargoRunner{}
			if cargoCmd := cargoRunner.findCommand(dir, "typecheck", r.Args); cargoCmd != nil {
				return r.ExecuteCommand(cargoCmd)
			}
		}

		// Go projects - use go build
		if FileExists(filepath.Join(dir, "go.mod")) {
			fmt.Fprintf(os.Stderr, "Running typecheck using go build...\n")
			goRunner := &goRunner{}
			if goCmd := goRunner.findCommand(dir, "typecheck", r.Args); goCmd != nil {
				return r.ExecuteCommand(goCmd)
			}
		}
	}

	return fmt.Errorf("could not synthesize typecheck command for this project")
}

// createTypescriptCheckCommand creates a TypeScript check command
func (r *CommandRunner) createTypescriptCheckCommand(dir string, packageManager string) *exec.Cmd {
	// Try to use tsc directly
	args := []string{"run", "tsc", "--noEmit"}
	args = append(args, r.Args...)
	cmd := exec.Command(packageManager, args...)
	cmd.Dir = dir
	return cmd
}
