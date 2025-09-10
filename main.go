package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/osteele/cmd-runner/internal"
)

const version = "0.1.0"

func showHelp() {
	fmt.Fprintf(os.Stderr, "cmd-runner %s - Smart command runner for multiple build systems\n\n", version)
	fmt.Fprintf(os.Stderr, "Usage: cmdr [OPTIONS] <command> [args...]\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  --interactive, -i       Launch interactive mode for command selection\n")
	fmt.Fprintf(os.Stderr, "  --list, -l              List available commands for current project\n")
	fmt.Fprintf(os.Stderr, "    --all                 Show commands from all sources (not just primary)\n")
	fmt.Fprintf(os.Stderr, "    --verbose             Show full command descriptions\n")
	fmt.Fprintf(os.Stderr, "  --version, -v           Show version information\n")
	fmt.Fprintf(os.Stderr, "  --help, -h              Show this help message\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Special Commands:\n")
	fmt.Fprintf(os.Stderr, "  install-alias [--dry-run]  Install 'cr' alias to shell config\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Common Commands:\n")
	fmt.Fprintf(os.Stderr, "  test       Run tests\n")
	fmt.Fprintf(os.Stderr, "  build      Build the project\n")
	fmt.Fprintf(os.Stderr, "  run        Run the project (or dev/serve)\n")
	fmt.Fprintf(os.Stderr, "  format     Format code (or fmt)\n")
	fmt.Fprintf(os.Stderr, "  lint       Run linters\n")
	fmt.Fprintf(os.Stderr, "  typecheck  Run type checker\n")
	fmt.Fprintf(os.Stderr, "  check      Run lint, typecheck, and test\n")
	fmt.Fprintf(os.Stderr, "  clean      Clean build artifacts\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Short Aliases:\n")
	fmt.Fprintf(os.Stderr, "  f → format    t → test     tc → typecheck\n")
	fmt.Fprintf(os.Stderr, "  r → run       s → serve    b  → build\n")
	fmt.Fprintf(os.Stderr, "  l → lint\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "See full documentation: https://github.com/osteele/cmd-runner\n")
}

func showVersion() {
	fmt.Printf("cmdr version %s\n", version)
}

func main() {
	// Parse arguments
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(1)
	}

	// Check for interactive mode first
	for _, arg := range os.Args[1:] {
		if arg == "--interactive" || arg == "-i" {
			if err := internal.RunInteractive(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	// Look for the first non-flag argument (the command)
	command := ""
	commandIndex := -1

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if !strings.HasPrefix(arg, "-") {
			command = arg
			commandIndex = i
			break
		}
	}

	// If no command found, process flags
	if command == "" {
		// Check for standalone flags
		for _, arg := range os.Args[1:] {
			switch arg {
			case "--help", "-h":
				showHelp()
				os.Exit(0)
			case "--version", "-v":
				showVersion()
				os.Exit(0)
			case "--list", "-l", "--commands":
				// Check if help is requested for list command
				for _, a := range os.Args[1:] {
					if a == "--help" || a == "-h" {
						fmt.Fprintf(os.Stderr, "Usage: cmdr --list [OPTIONS]\n")
						fmt.Fprintf(os.Stderr, "\n")
						fmt.Fprintf(os.Stderr, "List available commands for the current project.\n")
						fmt.Fprintf(os.Stderr, "\n")
						fmt.Fprintf(os.Stderr, "Options:\n")
						fmt.Fprintf(os.Stderr, "  --all, -a      Show commands from all sources (not just primary)\n")
						fmt.Fprintf(os.Stderr, "  --verbose      Show full command descriptions (no truncation)\n")
						fmt.Fprintf(os.Stderr, "  --help, -h     Show this help message\n")
						fmt.Fprintf(os.Stderr, "\n")
						fmt.Fprintf(os.Stderr, "By default, only commands from the primary source (e.g., mise, just, make)\n")
						fmt.Fprintf(os.Stderr, "are shown with descriptions truncated to fit the terminal width.\n")
						os.Exit(0)
					}
				}

				runner := internal.New("", nil)
				if err := runner.Init(); err != nil {
					fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
					os.Exit(1)
				}
				// Check for additional list options
				listAll := false
				verbose := false
				for _, a := range os.Args[1:] {
					if a == "--all" || a == "-a" || a == "--list-all" {
						listAll = true
					}
					if a == "--verbose" {
						verbose = true
					}
				}
				runner.ListCommandsWithOptions(listAll, verbose)
				os.Exit(0)
			default:
				if strings.HasPrefix(arg, "-") {
					fmt.Fprintf(os.Stderr, "Unknown option: %s\n", arg)
					fmt.Fprintf(os.Stderr, "Try 'cmdr --help' for more information.\n")
					os.Exit(1)
				}
			}
		}
		// No command and no recognized flags
		showHelp()
		os.Exit(1)
	}

	// We have a command - pass all args after it unchanged
	args := []string{}
	if commandIndex >= 0 && commandIndex+1 < len(os.Args) {
		args = os.Args[commandIndex+1:]
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
		if err := installAlias(dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error installing alias: %v\n", err)
			os.Exit(1)
		}
		return
	}

	runner := internal.New(command, args)

	if err := runner.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
		os.Exit(1)
	}

	if err := runner.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func installAlias(dryRun bool) error {
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

		if strings.Contains(string(content), aliasLine) {
			if dryRun {
				fmt.Printf("[DRY RUN] Alias 'cr' is already installed in %s\n", targetFile)
			} else {
				fmt.Printf("Alias 'cr' is already installed in %s\n", targetFile)
			}
			return nil
		}
	}

	if dryRun {
		fmt.Println("[DRY RUN] Would perform the following actions:")
		fmt.Printf("  - Add alias to: %s\n", targetFile)
		fmt.Printf("  - Add line: %s\n", aliasLine)
		if !internal.FileExists(targetFile) {
			fmt.Printf("  - Create new file: %s\n", targetFile)
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
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", closeErr)
		}
	}()

	// Add a newline before the alias to ensure it's on its own line
	_, err = fmt.Fprintf(file, "\n# Added by cmdr\n%s\n", aliasLine)
	if err != nil {
		return fmt.Errorf("failed to write to %s: %w", targetFile, err)
	}

	fmt.Printf("Successfully added 'cr' alias to %s\n", targetFile)
	fmt.Println("To use it immediately, run: source " + targetFile)
	fmt.Println("Or start a new terminal session.")
	return nil
}
