package internal

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// HandleTypecheckCommand handles the special 'typecheck' command
func HandleTypecheckCommand(r *CommandRunner) error {
	// First try to find a native typecheck command
	for _, project := range r.resolveProjects() {
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
	for _, project := range r.resolveProjects() {
		dir := project.Dir

		// TypeScript projects - use tsc
		if FileExists(filepath.Join(dir, "tsconfig.json")) {
			packageManager := nodePackageManagerName(project)
			fmt.Fprintf(r.stderrWriter(), "Running typecheck using tsc...\n")
			cmd := r.createTypescriptCheckCommand(dir, packageManager)
			if cmd != nil {
				return r.ExecuteCommand(cmd)
			}
		}

		// Python projects - try pyright or mypy
		if FileExists(filepath.Join(dir, "pyproject.toml")) {
			checker := pythonTypechecker(dir)
			if checker == "" {
				continue
			}

			// Detect if we have a Python package manager
			var packageManager string
			for _, source := range project.CommandSources {
				switch source.Name() {
				case "uv", "Poetry":
					packageManager = source.Name()
				}
			}

			var execCmd *exec.Cmd
			switch checker {
			case "pyright":
				switch packageManager {
				case "uv":
					cmdArgs := append([]string{"run", "pyright"}, r.Args...)
					execCmd = exec.Command("uv", cmdArgs...)
				case "Poetry":
					cmdArgs := append([]string{"run", "pyright"}, r.Args...)
					execCmd = exec.Command("poetry", cmdArgs...)
				default:
					execCmd = exec.Command("pyright", r.Args...)
				}
				fmt.Fprintf(r.stderrWriter(), "Running typecheck using pyright...\n")
			case "mypy":
				switch packageManager {
				case "uv":
					cmdArgs := append([]string{"run", "mypy", "."}, r.Args...)
					execCmd = exec.Command("uv", cmdArgs...)
				case "Poetry":
					cmdArgs := append([]string{"run", "mypy", "."}, r.Args...)
					execCmd = exec.Command("poetry", cmdArgs...)
				default:
					cmdArgs := append([]string{"."}, r.Args...)
					execCmd = exec.Command("mypy", cmdArgs...)
				}
				fmt.Fprintf(r.stderrWriter(), "Running typecheck using mypy...\n")
			}

			if execCmd != nil {
				execCmd.Dir = dir
				return r.ExecuteCommand(execCmd)
			}
		}

		// Rust projects - use cargo check
		if FileExists(filepath.Join(dir, "Cargo.toml")) {
			fmt.Fprintf(r.stderrWriter(), "Running typecheck using cargo check...\n")
			if cargoSource := findSourceByName(project.CommandSources, "Cargo"); cargoSource != nil {
				if cargoCmd := cargoSource.FindCommand("typecheck", r.Args); cargoCmd != nil {
					return r.ExecuteCommand(cargoCmd)
				}
			}
		}

		// Go projects - use go build
		if FileExists(filepath.Join(dir, "go.mod")) {
			fmt.Fprintf(r.stderrWriter(), "Running typecheck using go build...\n")
			if goSource := findSourceByName(project.CommandSources, "Go"); goSource != nil {
				if goCmd := goSource.FindCommand("typecheck", r.Args); goCmd != nil {
					return r.ExecuteCommand(goCmd)
				}
			}
		}
	}

	return fmt.Errorf("could not synthesize typecheck command for this project")
}

// createTypescriptCheckCommand creates a TypeScript check command
func (r *CommandRunner) createTypescriptCheckCommand(dir string, packageManager string) *exec.Cmd {
	var args []string
	var cmdName string

	switch packageManager {
	case "npm":
		// npm requires npx to run node_modules/.bin executables
		cmdName = "npx"
		args = append([]string{"tsc", "--noEmit"}, r.Args...)
	case "pnpm":
		// pnpm exec is the equivalent of npx
		cmdName = "pnpm"
		args = append([]string{"exec", "tsc", "--noEmit"}, r.Args...)
	case "yarn":
		// yarn run works for node_modules/.bin executables
		cmdName = "yarn"
		args = append([]string{"run", "tsc", "--noEmit"}, r.Args...)
	case "bun":
		// bun run works for node_modules/.bin executables
		cmdName = "bun"
		args = append([]string{"run", "tsc", "--noEmit"}, r.Args...)
	case "deno":
		// Deno has built-in type checking via "deno check"
		// Skip tsc entirely for Deno projects
		return nil
	default:
		// Fallback: try npx
		cmdName = "npx"
		args = append([]string{"tsc", "--noEmit"}, r.Args...)
	}

	cmd := exec.Command(cmdName, args...)
	cmd.Dir = dir
	return cmd
}
