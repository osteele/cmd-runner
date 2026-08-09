package internal

import (
	"os/exec"
	"path/filepath"
)

// Base Node.js source implementation
type nodeBaseSource struct {
	baseSource
	packageManager string
}

func (n *nodeBaseSource) ListCommands() map[string]CommandInfo {
	scripts, err := parsePackageJsonScripts(n.dir)
	if err != nil {
		return map[string]CommandInfo{}
	}

	commands := make(map[string]CommandInfo)
	for script, content := range scripts {
		execution := ""
		if n.packageManager == "deno" {
			execution = "deno task " + script
		} else {
			execution = n.packageManager + " run " + script
		}

		commands[script] = CommandInfo{
			Description: content,
			Execution:   execution,
		}
	}

	// Add standard commands if not in scripts
	if _, exists := commands["setup"]; !exists && n.packageManager != "deno" {
		commands["setup"] = CommandInfo{
			Description: "Install dependencies",
			Execution:   n.packageManager + " install",
		}
	}
	if _, exists := commands["install"]; !exists && n.packageManager != "deno" {
		linkCmd := "link"
		if n.packageManager == "pnpm" {
			linkCmd = "link --global"
		}
		commands["install"] = CommandInfo{
			Description: "Link binary globally",
			Execution:   n.packageManager + " " + linkCmd,
		}
	}

	return commands
}

func (n *nodeBaseSource) FindCommand(command string, args []string) *exec.Cmd {
	scripts, err := parsePackageJsonScripts(n.dir)
	if err != nil {
		return nil
	}

	_, scriptExists := scripts[command]

	// Special handling for setup command
	if !scriptExists && command == "setup" && n.packageManager != "deno" {
		cmdArgs := append([]string{"install"}, args...)
		cmd := exec.Command(n.packageManager, cmdArgs...)
		cmd.Dir = n.dir
		return cmd
	}

	// Special handling for install command (link binary globally)
	if !scriptExists && command == "install" && n.packageManager != "deno" {
		var cmdArgs []string
		if n.packageManager == "pnpm" {
			cmdArgs = append([]string{"link", "--global"}, args...)
		} else {
			cmdArgs = append([]string{"link"}, args...)
		}
		cmd := exec.Command(n.packageManager, cmdArgs...)
		cmd.Dir = n.dir
		return cmd
	}

	// Special handling for typecheck in TypeScript projects
	if !scriptExists && command == "typecheck" {
		if FileExists(filepath.Join(n.dir, "tsconfig.json")) {
			// Use tsc for TypeScript type checking with appropriate package manager syntax
			var cmdName string
			var cmdArgs []string

			switch n.packageManager {
			case "npm":
				// npm requires npx to run node_modules/.bin executables
				cmdName = "npx"
				cmdArgs = append([]string{"tsc", "--noEmit"}, args...)
			case "pnpm":
				// pnpm exec is the equivalent of npx
				cmdName = "pnpm"
				cmdArgs = append([]string{"exec", "tsc", "--noEmit"}, args...)
			case "yarn":
				// yarn run works for node_modules/.bin executables
				cmdName = "yarn"
				cmdArgs = append([]string{"run", "tsc", "--noEmit"}, args...)
			case "bun":
				// bun run works for node_modules/.bin executables
				cmdName = "bun"
				cmdArgs = append([]string{"run", "tsc", "--noEmit"}, args...)
			case "deno":
				// Deno projects should use "deno check" instead - skip tsc
				return nil
			default:
				// Fallback: try npx
				cmdName = "npx"
				cmdArgs = append([]string{"tsc", "--noEmit"}, args...)
			}

			cmd := exec.Command(cmdName, cmdArgs...)
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

	// npm requires -- separator to pass arguments to scripts
	var cmdArgs []string
	if n.packageManager == "npm" && len(args) > 0 {
		cmdArgs = append([]string{"run", command, "--"}, args...)
	} else {
		cmdArgs = append([]string{"run", command}, args...)
	}
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

func (d *DenoSource) ListCommands() map[string]CommandInfo {
	commands := make(map[string]CommandInfo)

	for task, description := range d.listTasks() {
		commands[task] = CommandInfo{
			Description: description,
			Execution:   "deno task " + task,
		}
	}

	// Add standard Deno commands
	builtins := map[string]CommandInfo{
		"run":       {Description: "Run a script", Execution: "deno run"},
		"test":      {Description: "Run tests", Execution: "deno test"},
		"lint":      {Description: "Run linter", Execution: "deno lint"},
		"format":    {Description: "Format code", Execution: "deno fmt"},
		"typecheck": {Description: "Type-check code", Execution: "deno check"},
		"check":     {Description: "Type-check code", Execution: "deno check"},
		"build":     {Description: "Compile to executable", Execution: "deno compile"},
	}
	for command, info := range builtins {
		if _, exists := commands[command]; !exists {
			commands[command] = info
		}
	}

	return commands
}

func (d *DenoSource) FindCommand(command string, args []string) *exec.Cmd {
	if _, exists := d.listTasks()[command]; exists {
		cmdArgs := append([]string{"task", command}, args...)
		cmd := exec.Command("deno", cmdArgs...)
		cmd.Dir = d.dir
		return cmd
	}

	// Deno built-in commands
	denoCommands := map[string]string{
		"run":       "run",
		"test":      "test",
		"lint":      "lint",
		"format":    "fmt",
		"typecheck": "check",
		"check":     "check",
		"build":     "compile",
		"install":   "install",
	}

	if denoCmd, ok := denoCommands[command]; ok {
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

	return nil
}

func (d *DenoSource) listTasks() map[string]string {
	tasks := make(map[string]string)
	if FileExists(filepath.Join(d.dir, "package.json")) {
		if scripts, err := parsePackageJsonScripts(d.dir); err == nil {
			for name, command := range scripts {
				tasks[name] = command
			}
		}
	}
	for _, filename := range []string{"deno.json", "deno.jsonc"} {
		var config struct {
			Tasks map[string]string `json:"tasks"`
		}
		if decodeJSONCFile(filepath.Join(d.dir, filename), &config) == nil {
			for name, command := range config.Tasks {
				tasks[name] = command
			}
			return tasks
		}
	}
	return tasks
}
