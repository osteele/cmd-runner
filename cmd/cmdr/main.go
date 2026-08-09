package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/osteele/cmd-runner/internal"
)

var runInteractive = internal.RunInteractive

func showHelp(stderr io.Writer) {
	fmt.Fprintf(stderr, "cmd-runner %s - Smart command runner for multiple build systems\n\n", currentVersion())
	fmt.Fprintln(stderr, "Usage: cmdr [OPTIONS] [command] [args...]")
	fmt.Fprintln(stderr)
	fmt.Fprintln(stderr, "When run without arguments, shows available commands (same as --list).")
	fmt.Fprintln(stderr)
	fmt.Fprintln(stderr, "Options:")
	fmt.Fprintln(stderr, "  --interactive, -i       Launch interactive mode for command selection")
	fmt.Fprintln(stderr, "  --list, -l              List available commands for current project")
	fmt.Fprintln(stderr, "    --all                 Show commands from all sources (not just primary)")
	fmt.Fprintln(stderr, "    --verbose             Show full descriptions and configuration warnings")
	fmt.Fprintln(stderr, "  --version, -v           Show version information")
	fmt.Fprintln(stderr, "  --help, -h              Show this help message")
	fmt.Fprintln(stderr)
	fmt.Fprintln(stderr, "Special Commands:")
	fmt.Fprintln(stderr, "  install-alias [--dry-run]  Install 'cr' alias to shell config")
	fmt.Fprintln(stderr)
	fmt.Fprintln(stderr, "Common Commands:")
	fmt.Fprintln(stderr, "  setup      Install dependencies for local development")
	fmt.Fprintln(stderr, "  install    Install binary/package globally")
	fmt.Fprintln(stderr, "  test       Run tests")
	fmt.Fprintln(stderr, "  build      Build the project")
	fmt.Fprintln(stderr, "  run        Run the project (or dev/serve)")
	fmt.Fprintln(stderr, "  format     Format code (or fmt)")
	fmt.Fprintln(stderr, "  lint       Run linters")
	fmt.Fprintln(stderr, "  typecheck  Run type checker")
	fmt.Fprintln(stderr, "  check      Run lint, typecheck, and test")
	fmt.Fprintln(stderr, "  clean      Clean build artifacts")
	fmt.Fprintln(stderr)
	fmt.Fprintln(stderr, "Short Aliases:")
	fmt.Fprintln(stderr, "  f → format    t → test     tc → typecheck")
	fmt.Fprintln(stderr, "  r → run       s → serve    b  → build")
	fmt.Fprintln(stderr, "  l → lint")
	fmt.Fprintln(stderr)
	fmt.Fprintln(stderr, "See full documentation: https://github.com/osteele/cmd-runner")
}

func showVersion(stdout io.Writer) {
	fmt.Fprintf(stdout, "cmdr version %s\n", currentVersion())
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(cliArgs []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// Parse arguments
	if len(cliArgs) == 0 {
		// No arguments - show command list
		runner := internal.New("", nil)
		runner.Stdin, runner.Stdout, runner.Stderr = stdin, stdout, stderr
		if err := runner.Init(); err != nil {
			fmt.Fprintf(stderr, "Error initializing: %v\n", err)
			return 1
		}
		runner.ListCommands()
		return 0
	}

	preCommandFlags := []string{}
	command := ""
	commandIndex := -1

	for i, arg := range cliArgs {
		if arg == "--" {
			if i+1 < len(cliArgs) {
				command = cliArgs[i+1]
				commandIndex = i + 1
			}
			break
		}
		if strings.HasPrefix(arg, "-") && command == "" {
			preCommandFlags = append(preCommandFlags, arg)
			continue
		}
		if command == "" {
			command = arg
			commandIndex = i
		}
		break
	}

	listRequested := false
	for _, flag := range preCommandFlags {
		if flag == "--list" || flag == "-l" || flag == "--commands" {
			listRequested = true
		}
	}

	listAll := false
	verbose := false
	showHelpFlag := false

	for _, flag := range preCommandFlags {
		switch flag {
		case "--interactive", "-i":
			if err := runInteractive(); err != nil {
				fmt.Fprintf(stderr, "Error: %v\n", err)
				return 1
			}
			return 0
		case "--help", "-h":
			showHelpFlag = true
		case "--version", "-v":
			showVersion(stdout)
			return 0
		case "--list", "-l", "--commands":
			// processed after loop
		case "--all", "-a", "--list-all":
			if !listRequested {
				fmt.Fprintf(stderr, "Unknown option: %s\n", flag)
				fmt.Fprintln(stderr, "Try 'cmdr --help' for more information.")
				return 1
			}
			listAll = true
		case "--verbose":
			if !listRequested {
				fmt.Fprintf(stderr, "Unknown option: %s\n", flag)
				fmt.Fprintln(stderr, "Try 'cmdr --help' for more information.")
				return 1
			}
			verbose = true
		default:
			fmt.Fprintf(stderr, "Unknown option: %s\n", flag)
			fmt.Fprintln(stderr, "Try 'cmdr --help' for more information.")
			return 1
		}
	}

	if showHelpFlag && !listRequested {
		showHelp(stderr)
		return 0
	}

	if listRequested && command == "" {
		if showHelpFlag {
			fmt.Fprintln(stderr, "Usage: cmdr --list [OPTIONS]")
			fmt.Fprintln(stderr)
			fmt.Fprintln(stderr, "List available commands for the current project.")
			fmt.Fprintln(stderr)
			fmt.Fprintln(stderr, "Options:")
			fmt.Fprintln(stderr, "  --all, -a      Show commands from all sources (not just primary)")
			fmt.Fprintln(stderr, "  --verbose      Show full descriptions and configuration warnings")
			fmt.Fprintln(stderr, "  --help, -h     Show this help message")
			fmt.Fprintln(stderr)
			fmt.Fprintln(stderr, "By default, only commands from the primary source (e.g., mise, just, make)")
			fmt.Fprintln(stderr, "are shown with descriptions truncated to fit the terminal width.")
			return 0
		}

		runner := internal.New("", nil)
		runner.Stdin, runner.Stdout, runner.Stderr = stdin, stdout, stderr
		if err := runner.Init(); err != nil {
			fmt.Fprintf(stderr, "Error initializing: %v\n", err)
			return 1
		}
		runner.ListCommandsWithOptions(listAll, verbose)
		return 0
	}

	if listRequested && command != "" {
		fmt.Fprintln(stderr, "The --list flag must appear without a command.")
		return 1
	}

	if command == "" {
		if showHelpFlag {
			showHelp(stderr)
			return 0
		}
		showHelp(stderr)
		return 1
	}

	// We have a command - pass all args after it unchanged
	args := []string{}
	if commandIndex >= 0 && commandIndex+1 < len(cliArgs) {
		args = cliArgs[commandIndex+1:]
	}

	// Handle special commands
	if command == "install-alias" {
		dryRun := false
		for _, arg := range args {
			if arg == "--dry-run" || arg == "-n" {
				dryRun = true
				break
			}
		}
		if err := installAlias(dryRun, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "Error installing alias: %v\n", err)
			return 1
		}
		return 0
	}

	runner := internal.New(command, args)
	runner.Stdin, runner.Stdout, runner.Stderr = stdin, stdout, stderr

	if err := runner.Init(); err != nil {
		fmt.Fprintf(stderr, "Error initializing: %v\n", err)
		return 1
	}

	if err := runner.Run(); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func installAlias(dryRun bool, stdout, stderr io.Writer) error {
	// Determine which shell config file to use
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check shell and determine config file
	shell := os.Getenv("SHELL")
	var configFiles []string

	if strings.Contains(shell, "zsh") {
		configFiles = []string{
			filepath.Join(homeDir, ".zshrc"),
			filepath.Join(homeDir, ".zprofile"),
		}
	} else if strings.Contains(shell, "bash") {
		configFiles = []string{
			filepath.Join(homeDir, ".bashrc"),
			filepath.Join(homeDir, ".bash_profile"),
			filepath.Join(homeDir, ".profile"),
		}
	} else {
		// Default to common shell config files
		configFiles = []string{
			filepath.Join(homeDir, ".zshrc"),
			filepath.Join(homeDir, ".bashrc"),
			filepath.Join(homeDir, ".profile"),
		}
	}

	aliasLine := "alias cr=cmdr"

	// Find the first existing config file
	var targetFile string
	for _, file := range configFiles {
		if internal.FileExists(file) {
			targetFile = file
			break
		}
	}

	// If no config file exists, create the most appropriate one
	if targetFile == "" {
		if strings.Contains(shell, "zsh") {
			targetFile = filepath.Join(homeDir, ".zshrc")
		} else {
			targetFile = filepath.Join(homeDir, ".bashrc")
		}
	}

	// Check if alias already exists
	if internal.FileExists(targetFile) {
		content, err := os.ReadFile(targetFile)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", targetFile, err)
		}

		// Match "alias cr=cmdr" as a complete statement (not substring)
		// This avoids false positives like "alias cr=cmdr-dev" or commented lines
		aliasPattern := regexp.MustCompile(`(?m)^\s*alias\s+cr=cmdr\s*$`)
		if aliasPattern.Match(content) {
			if dryRun {
				fmt.Fprintf(stdout, "[DRY RUN] Alias 'cr' is already installed in %s\n", targetFile)
			} else {
				fmt.Fprintf(stdout, "Alias 'cr' is already installed in %s\n", targetFile)
			}
			return nil
		}
	}

	if dryRun {
		fmt.Fprintln(stdout, "[DRY RUN] Would perform the following actions:")
		fmt.Fprintf(stdout, "  - Add alias to: %s\n", targetFile)
		fmt.Fprintf(stdout, "  - Add line: %s\n", aliasLine)
		if !internal.FileExists(targetFile) {
			fmt.Fprintf(stdout, "  - Create new file: %s\n", targetFile)
		}
		return nil
	}

	// Append alias to config file
	file, err := os.OpenFile(targetFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", targetFile, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(stderr, "Warning: failed to close file: %v\n", closeErr)
		}
	}()

	// Add a newline before the alias to ensure it's on its own line
	_, err = fmt.Fprintf(file, "\n# Added by cmdr\n%s\n", aliasLine)
	if err != nil {
		return fmt.Errorf("failed to write to %s: %w", targetFile, err)
	}

	fmt.Fprintf(stdout, "Successfully added 'cr' alias to %s\n", targetFile)
	fmt.Fprintln(stdout, "To use it immediately, run: source "+targetFile)
	fmt.Fprintln(stdout, "Or start a new terminal session.")
	return nil
}
