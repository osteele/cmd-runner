package cmdrunner

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MiseSource represents commands from .mise.toml
type MiseSource struct {
	baseSource
}

func NewMiseSource(dir string) CommandSource {
	return &MiseSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "mise",
			priority: 1,
		},
	}
}

func (m *MiseSource) ListCommands() map[string]CommandInfo {
	commands := make(map[string]CommandInfo)

	testCmd := exec.Command("mise", "tasks", "ls")
	testCmd.Dir = m.dir
	if output, err := testCmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				// mise tasks ls outputs: "taskname  description" or just "taskname"
				// Split on whitespace to separate task name from description
				parts := strings.Fields(line)
				if len(parts) > 0 {
					taskName := parts[0]
					description := ""
					if len(parts) > 1 {
						// Join the rest as the description
						description = strings.Join(parts[1:], " ")
					}
					commands[taskName] = CommandInfo{
						Description: description,
						Execution:   "mise run " + taskName,
					}
				}
			}
		}
	}

	return commands
}

func (m *MiseSource) FindCommand(command string, args []string) *exec.Cmd {
	// Check if the command exists
	for _, variant := range GetCommandVariants(command) {
		testCmd := exec.Command("mise", "tasks", "ls")
		testCmd.Dir = m.dir
		output, err := testCmd.Output()
		if err == nil && strings.Contains(string(output), variant) {
			cmdArgs := append([]string{"run", variant}, args...)
			cmd := exec.Command("mise", cmdArgs...)
			cmd.Dir = m.dir
			return cmd
		}
	}
	return nil
}

// JustSource represents commands from justfile
type JustSource struct {
	baseSource
}

func NewJustSource(dir string) CommandSource {
	return &JustSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "just",
			priority: 2,
		},
	}
}

func (j *JustSource) ListCommands() map[string]CommandInfo {
	commands := make(map[string]CommandInfo)

	testCmd := exec.Command("just", "--list")
	testCmd.Dir = j.dir
	if output, err := testCmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "Available") {
				// just output format: "command   # description"
				parts := strings.SplitN(line, "#", 2)
				if len(parts) > 0 {
					cmd := strings.TrimSpace(parts[0])
					desc := ""
					if len(parts) > 1 {
						desc = strings.TrimSpace(parts[1])
					}
					if cmd != "" {
						commands[cmd] = CommandInfo{
							Description: desc,
							Execution:   "just " + cmd,
						}
					}
				}
			}
		}
	}

	return commands
}

func (j *JustSource) FindCommand(command string, args []string) *exec.Cmd {
	for _, variant := range GetCommandVariants(command) {
		testCmd := exec.Command("just", "--list")
		testCmd.Dir = j.dir
		output, err := testCmd.Output()
		if err == nil && strings.Contains(string(output), variant) {
			cmdArgs := append([]string{variant}, args...)
			cmd := exec.Command("just", cmdArgs...)
			cmd.Dir = j.dir
			return cmd
		}
	}
	return nil
}

// MakeSource represents commands from Makefile
type MakeSource struct {
	baseSource
}

func NewMakeSource(dir string) CommandSource {
	return &MakeSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "make",
			priority: 3,
		},
	}
}

func (m *MakeSource) ListCommands() map[string]CommandInfo {
	commands := make(map[string]CommandInfo)

	makefiles := []string{"Makefile", "makefile"}
	for _, mf := range makefiles {
		path := filepath.Join(m.dir, mf)
		if FileExists(path) {
			file, err := os.Open(path)
			if err != nil {
				continue
			}
			defer func() {
				_ = file.Close()
			}()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				// Look for targets (lines ending with :)
				if strings.Contains(line, ":") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") {
					parts := strings.Split(line, ":")
					if len(parts) > 0 {
						target := strings.TrimSpace(parts[0])
						// Skip special targets and variables
						if !strings.HasPrefix(target, ".") && !strings.Contains(target, "=") && target != "" {
							commands[target] = CommandInfo{
								Description: target,
								Execution:   "make " + target,
							}
						}
					}
				}
			}
		}
	}

	return commands
}

func (m *MakeSource) FindCommand(command string, args []string) *exec.Cmd {
	for _, variant := range GetCommandVariants(command) {
		if m.hasTarget(variant) {
			cmdArgs := append([]string{variant}, args...)
			cmd := exec.Command("make", cmdArgs...)
			cmd.Dir = m.dir
			return cmd
		}
	}
	return nil
}

func (m *MakeSource) hasTarget(target string) bool {
	makefiles := []string{"Makefile", "makefile"}
	for _, mf := range makefiles {
		path := filepath.Join(m.dir, mf)
		if FileExists(path) {
			file, err := os.Open(path)
			if err != nil {
				continue
			}
			defer func() {
				_ = file.Close()
			}()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, target+":") {
					return true
				}
			}
		}
	}
	return false
}
