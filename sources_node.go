package cmdrunner

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// Base Node.js source implementation
type nodeBaseSource struct {
	baseSource
	packageManager string
}

func (n *nodeBaseSource) ListCommands() map[string]string {
	commands := make(map[string]string)
	
	scripts, err := parsePackageJsonScripts(n.dir)
	if err != nil {
		return commands
	}
	
	for script := range scripts {
		if n.packageManager == "deno" {
			commands[script] = "deno task " + script
		} else {
			commands[script] = n.packageManager + " run " + script
		}
	}
	
	return commands
}

func (n *nodeBaseSource) FindCommand(command string, args []string) *exec.Cmd {
	scripts, err := parsePackageJsonScripts(n.dir)
	if err != nil {
		return nil
	}
	
	// Check if script exists
	var scriptExists bool
	for _, variant := range GetCommandVariants(command) {
		if _, ok := scripts[variant]; ok {
			command = variant
			scriptExists = true
			break
		}
	}
	
	// Special handling for typecheck in TypeScript projects
	if !scriptExists && command == "typecheck" {
		if FileExists(filepath.Join(n.dir, "tsconfig.json")) {
			// Use tsc for TypeScript type checking
			cmdArgs := append([]string{"run", "tsc", "--noEmit"}, args...)
			cmd := exec.Command(n.packageManager, cmdArgs...)
			cmd.Dir = n.dir
			return cmd
		}
	}
	
	if !scriptExists {
		return nil
	}
	
	// Deno uses "task" instead of "run"
	if n.packageManager == "deno" {
		cmdArgs := append([]string{"task", command}, args...)
		cmd := exec.Command(n.packageManager, cmdArgs...)
		cmd.Dir = n.dir
		return cmd
	}
	
	cmdArgs := append([]string{"run", command}, args...)
	cmd := exec.Command(n.packageManager, cmdArgs...)
	cmd.Dir = n.dir
	return cmd
}

// NpmSource for npm projects
type NpmSource struct {
	nodeBaseSource
}

func NewNpmSource(dir string) CommandSource {
	return &NpmSource{
		nodeBaseSource: nodeBaseSource{
			baseSource: baseSource{
				dir:      dir,
				name:     "npm",
				priority: 10,
			},
			packageManager: "npm",
		},
	}
}

// BunSource for bun projects
type BunSource struct {
	nodeBaseSource
}

func NewBunSource(dir string) CommandSource {
	return &BunSource{
		nodeBaseSource: nodeBaseSource{
			baseSource: baseSource{
				dir:      dir,
				name:     "bun",
				priority: 10,
			},
			packageManager: "bun",
		},
	}
}

// PnpmSource for pnpm projects
type PnpmSource struct {
	nodeBaseSource
}

func NewPnpmSource(dir string) CommandSource {
	return &PnpmSource{
		nodeBaseSource: nodeBaseSource{
			baseSource: baseSource{
				dir:      dir,
				name:     "pnpm",
				priority: 10,
			},
			packageManager: "pnpm",
		},
	}
}

// YarnSource for yarn projects
type YarnSource struct {
	nodeBaseSource
}

func NewYarnSource(dir string) CommandSource {
	return &YarnSource{
		nodeBaseSource: nodeBaseSource{
			baseSource: baseSource{
				dir:      dir,
				name:     "yarn",
				priority: 10,
			},
			packageManager: "yarn",
		},
	}
}

// DenoSource for Deno projects
type DenoSource struct {
	baseSource
}

func NewDenoSource(dir string) CommandSource {
	return &DenoSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "Deno",
			priority: 10,
		},
	}
}

func (d *DenoSource) ListCommands() map[string]string {
	commands := make(map[string]string)
	
	// Check if there's a package.json (Deno can use it)
	if FileExists(filepath.Join(d.dir, "package.json")) {
		if scripts, err := parsePackageJsonScripts(d.dir); err == nil {
			for script := range scripts {
				commands[script] = "deno task " + script
			}
		}
	}
	
	// Add standard Deno commands
	commands["run"] = "deno run"
	commands["test"] = "deno test"
	commands["lint"] = "deno lint"
	commands["format"] = "deno fmt"
	commands["check"] = "deno check"
	commands["build"] = "deno compile"
	
	return commands
}

func (d *DenoSource) FindCommand(command string, args []string) *exec.Cmd {
	// Deno built-in commands
	denoCommands := map[string]string{
		"run":       "run",
		"dev":       "run",
		"start":     "run",
		"test":      "test",
		"lint":      "lint",
		"format":    "fmt",
		"fmt":       "fmt",
		"typecheck": "check",
		"tc":        "check",
		"check":     "check",
		"build":     "compile",
		"install":   "install",
	}
	
	for _, variant := range GetCommandVariants(command) {
		if denoCmd, ok := denoCommands[variant]; ok {
			// For run commands, try to find the main file
			if denoCmd == "run" {
				// Look for common entry points
				for _, entry := range []string{"main.ts", "main.js", "mod.ts", "mod.js", "index.ts", "index.js"} {
					if FileExists(filepath.Join(d.dir, entry)) {
						cmdArgs := append([]string{"run", "--allow-all", entry}, args...)
						cmd := exec.Command("deno", cmdArgs...)
						cmd.Dir = d.dir
						return cmd
					}
				}
			}
			cmdArgs := append([]string{denoCmd}, args...)
			cmd := exec.Command("deno", cmdArgs...)
			cmd.Dir = d.dir
			return cmd
		}
	}
	
	// Check if there's a task defined in deno.json
	if FileExists(filepath.Join(d.dir, "deno.json")) || FileExists(filepath.Join(d.dir, "deno.jsonc")) {
		for _, variant := range GetCommandVariants(command) {
			// Try to run as a task
			testCmd := exec.Command("deno", "task", "--list")
			testCmd.Dir = d.dir
			output, err := testCmd.Output()
			if err == nil && strings.Contains(string(output), variant) {
				cmdArgs := append([]string{"task", variant}, args...)
				cmd := exec.Command("deno", cmdArgs...)
				cmd.Dir = d.dir
				return cmd
			}
		}
	}
	
	return nil
}