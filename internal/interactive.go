package internal

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// InteractiveSession manages the interactive mode state
type InteractiveSession struct {
	runner            *CommandRunner
	terminal          *TerminalManager
	lastCommand       string
	lastExitCode      int
	viewingOutput     bool
	availableCommands map[string]CommandInfo
	commandShortcuts  map[rune]string
	numberCommands    []string
}

// RunInteractive starts the interactive command runner mode
func RunInteractive() error {
	runner := New("", nil)
	if err := runner.Init(); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	session := &InteractiveSession{
		runner:           runner,
		terminal:         NewTerminalManager(),
		commandShortcuts: make(map[rune]string),
		numberCommands:   make([]string, 0),
	}

	// Setup cleanup on exit
	defer session.cleanup()
	session.terminal.SetupSignalHandling(session.cleanup)

	// Gather available commands
	session.gatherCommands()

	// Main interactive loop
	for {
		if session.viewingOutput {
			if err := session.showOutputView(); err != nil {
				if err.Error() == "quit" {
					return nil
				}
			}
		} else {
			if err := session.showMenu(); err != nil {
				if err.Error() == "quit" {
					return nil
				}
			}
		}
	}
}

// gatherCommands collects all available commands and sets up shortcuts
func (s *InteractiveSession) gatherCommands() {
	s.availableCommands = make(map[string]CommandInfo)

	// Build projects for current dir and project root
	projects := []*Project{}
	projects = append(projects, ResolveProject(s.runner.CurrentDir))

	if s.runner.ProjectRoot != s.runner.CurrentDir && s.runner.ProjectRoot != "" {
		projects = append(projects, ResolveProject(s.runner.ProjectRoot))
	}

	// Collect commands from all sources
	for _, project := range projects {
		for _, source := range project.CommandSources {
			commands := source.ListCommands()
			for cmd, info := range commands {
				if !isPrivateCommand(cmd) {
					s.availableCommands[cmd] = info
				}
			}
		}
	}

	// Add synthesized commands
	synth := map[string]CommandInfo{
		"check": {Description: "Runs lint, typecheck, and test", Execution: "synthesized"},
		"fix":   {Description: "Runs format and lint fix", Execution: "synthesized"},
	}

	// Only add typecheck if project has capability
	if s.runner.hasTypecheckCapability() {
		synth["typecheck"] = CommandInfo{Description: "Runs type checking", Execution: "synthesized"}
	}

	for cmd, info := range synth {
		if _, exists := s.availableCommands[cmd]; !exists {
			s.availableCommands[cmd] = info
		}
	}

	// Setup shortcuts for common commands
	shortcuts := map[rune]string{
		't': "test",
		'b': "build",
		'r': "run",
		'f': "format",
		'l': "lint",
		'c': "check",
		'x': "fix",
		's': "serve",
	}

	// Only add shortcuts for commands that exist
	for key, cmd := range shortcuts {
		if _, exists := s.availableCommands[cmd]; exists {
			s.commandShortcuts[key] = cmd
		}
	}

	// Setup number shortcuts for other commands
	otherCommands := make([]string, 0)
	for cmd := range s.availableCommands {
		isShortcut := false
		for _, shortcutCmd := range s.commandShortcuts {
			if cmd == shortcutCmd {
				isShortcut = true
				break
			}
		}
		if !isShortcut {
			otherCommands = append(otherCommands, cmd)
		}
	}

	sort.Strings(otherCommands)
	s.numberCommands = otherCommands
}

// showMenu displays the interactive menu
func (s *InteractiveSession) showMenu() error {
	fmt.Println("\ncmd-runner interactive mode")
	fmt.Println("─────────────────────────────────────")
	fmt.Println("Available commands:")
	fmt.Println()

	// Show common commands with shortcuts
	if len(s.commandShortcuts) > 0 {
		fmt.Println("Common:")
		commonCmds := []struct {
			key rune
			cmd string
		}{
			{'t', "test"}, {'b', "build"}, {'r', "run"},
			{'f', "format"}, {'l', "lint"}, {'c', "check"},
			{'x', "fix"}, {'s', "serve"},
		}

		for i, item := range commonCmds {
			if cmd, exists := s.commandShortcuts[item.key]; exists {
				fmt.Printf("  [%c] %-10s", item.key, cmd)
				if (i+1)%3 == 0 {
					fmt.Println()
				}
			}
		}
		fmt.Println()
	}

	// Show other commands with number shortcuts
	if len(s.numberCommands) > 0 {
		fmt.Println("\nOther commands:")
		for i, cmd := range s.numberCommands {
			if i >= 9 {
				break // Only show first 9
			}
			fmt.Printf("  [%d] %-10s", i+1, cmd)
			if (i+1)%3 == 0 {
				fmt.Println()
			}
		}
		if len(s.numberCommands) > 9 {
			fmt.Printf("\n  ... and %d more", len(s.numberCommands)-9)
		}
		fmt.Println()
	}

	// Show controls
	fmt.Println()
	if s.lastCommand != "" {
		status := "✓"
		if s.lastExitCode != 0 {
			status = "✗"
		}
		fmt.Printf("[.] repeat (%s %s)  ", s.lastCommand, status)
		fmt.Printf("[/] toggle output  ")
	}
	fmt.Println("[q] quit  [?] help")
	fmt.Println()
	fmt.Print("Select command (or type name): ")

	// Read user input
	if err := s.terminal.SetRawMode(); err != nil {
		return err
	}
	defer func() {
		_ = s.terminal.RestoreMode()
	}()

	key, err := s.terminal.ReadKey()
	if err != nil {
		if err.Error() == "interrupt" {
			return fmt.Errorf("quit")
		}
		return err
	}

	// Handle menu commands
	switch key {
	case 'q', 'Q':
		return fmt.Errorf("quit")
	case '.':
		if s.lastCommand != "" {
			return s.runCommand(s.lastCommand)
		}
	case '/':
		if s.lastCommand != "" {
			s.viewingOutput = true
			return nil
		}
	case '?':
		s.showHelp()
		return nil
	default:
		// Check if it's a shortcut
		if cmd, exists := s.commandShortcuts[key]; exists {
			return s.runCommand(cmd)
		}

		// Check if it's a number
		if key >= '1' && key <= '9' {
			index := int(key - '1')
			if index < len(s.numberCommands) {
				return s.runCommand(s.numberCommands[index])
			}
		}

		// Otherwise, enter type mode
		_ = s.terminal.RestoreMode()
		return s.typeMode(key)
	}

	return nil
}

// showOutputView shows the last command output
func (s *InteractiveSession) showOutputView() error {
	fmt.Println("\n[Last command output]")
	fmt.Println("─────────────────────────────────────")
	fmt.Printf("Command: %s (exit code: %d)\n", s.lastCommand, s.lastExitCode)
	fmt.Println("─────────────────────────────────────")
	fmt.Println()
	fmt.Println("Press '/' to return to menu, 'q' to quit")

	// Read user input
	if err := s.terminal.SetRawMode(); err != nil {
		return err
	}
	defer func() {
		_ = s.terminal.RestoreMode()
	}()

	key, err := s.terminal.ReadKey()
	if err != nil {
		if err.Error() == "interrupt" {
			return fmt.Errorf("quit")
		}
		return err
	}

	switch key {
	case 'q', 'Q':
		return fmt.Errorf("quit")
	case '/':
		s.viewingOutput = false
		return nil
	}

	return nil
}

// typeMode allows typing command names
func (s *InteractiveSession) typeMode(firstKey rune) error {
	fmt.Printf("\rType command name: %c", firstKey)

	// Read the rest of the command name
	input := string(firstKey)
	var line string
	_, _ = fmt.Scanln(&line)
	input += line

	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	// Find matching command
	if _, exists := s.availableCommands[input]; exists {
		return s.runCommand(input)
	}

	// Try to find partial match
	var matches []string
	for cmd := range s.availableCommands {
		if strings.HasPrefix(cmd, input) {
			matches = append(matches, cmd)
		}
	}

	if len(matches) == 1 {
		return s.runCommand(matches[0])
	} else if len(matches) > 1 {
		fmt.Printf("\nMultiple matches found: %s\n", strings.Join(matches, ", "))
		fmt.Println("Press any key to continue...")
		_ = s.terminal.SetRawMode()
		_, _ = s.terminal.ReadKey()
		_ = s.terminal.RestoreMode()
	} else {
		fmt.Printf("\nCommand '%s' not found\n", input)
		fmt.Println("Press any key to continue...")
		_ = s.terminal.SetRawMode()
		_, _ = s.terminal.ReadKey()
		_ = s.terminal.RestoreMode()
	}

	return nil
}

// runCommand executes a command and returns to menu
func (s *InteractiveSession) runCommand(command string) error {
	_ = s.terminal.RestoreMode()

	fmt.Println()
	fmt.Println("─────────────────────────────────────")
	fmt.Printf("Running: %s\n", command)
	fmt.Println("─────────────────────────────────────")

	// Create a new runner for this command
	runner := New(command, nil)
	if err := runner.Init(); err != nil {
		return err
	}

	// Run the command
	err := runner.Run()

	// Store last command info
	s.lastCommand = command
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			s.lastExitCode = exitErr.ExitCode()
		} else {
			s.lastExitCode = 1
		}
	} else {
		s.lastExitCode = 0
	}

	// Show completion status
	fmt.Println("─────────────────────────────────────")
	if s.lastExitCode == 0 {
		fmt.Printf("Command completed successfully\n")
		// Auto-return to menu on success
	} else {
		fmt.Printf("Command failed (exit code: %d)\n", s.lastExitCode)
		fmt.Println("Press any key to continue...")

		// Wait for keypress on failure
		_ = s.terminal.SetRawMode()
		_, _ = s.terminal.ReadKey()
		_ = s.terminal.RestoreMode()
	}

	return nil
}

// showHelp displays help information
func (s *InteractiveSession) showHelp() {
	_ = s.terminal.RestoreMode()

	fmt.Println("\nInteractive Mode Help")
	fmt.Println("─────────────────────────────────────")
	fmt.Println("Shortcuts:")
	fmt.Println("  t - test       b - build     r - run")
	fmt.Println("  f - format     l - lint      c - check")
	fmt.Println("  x - fix        s - serve")
	fmt.Println()
	fmt.Println("Controls:")
	fmt.Println("  1-9 - Run numbered command")
	fmt.Println("  .   - Repeat last command")
	fmt.Println("  /   - Toggle between menu and last output")
	fmt.Println("  q   - Quit interactive mode")
	fmt.Println("  ?   - Show this help")
	fmt.Println()
	fmt.Println("You can also type the full command name")
	fmt.Println("or a unique prefix to run it.")
	fmt.Println()
	fmt.Println("Press any key to continue...")

	_ = s.terminal.SetRawMode()
	_, _ = s.terminal.ReadKey()
	_ = s.terminal.RestoreMode()
}

// cleanup restores terminal state
func (s *InteractiveSession) cleanup() {
	_ = s.terminal.RestoreMode()
	fmt.Println("\nGoodbye!")
}
