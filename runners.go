package cmdrunner

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type commandFinder interface {
	findCommand(dir, command string, args []string) *exec.Cmd
}

type miseRunner struct{}

func (m *miseRunner) findCommand(dir, command string, args []string) *exec.Cmd {
	miseFile := filepath.Join(dir, ".mise.toml")
	if !FileExists(miseFile) {
		return nil
	}

	for _, variant := range GetCommandVariants(command) {
		testCmd := exec.Command("mise", "run", "--list")
		testCmd.Dir = dir
		output, err := testCmd.Output()
		if err == nil && strings.Contains(string(output), variant) {
			cmdArgs := append([]string{"run", variant}, args...)
			return exec.Command("mise", cmdArgs...)
		}
	}
	return nil
}

type justRunner struct{}

func (j *justRunner) findCommand(dir, command string, args []string) *exec.Cmd {
	justfile := filepath.Join(dir, "justfile")
	if !FileExists(justfile) && !FileExists(filepath.Join(dir, "Justfile")) {
		return nil
	}

	for _, variant := range GetCommandVariants(command) {
		testCmd := exec.Command("just", "--list")
		testCmd.Dir = dir
		output, err := testCmd.Output()
		if err == nil && strings.Contains(string(output), variant) {
			cmdArgs := append([]string{variant}, args...)
			return exec.Command("just", cmdArgs...)
		}
	}
	return nil
}

type makeRunner struct{}

func (m *makeRunner) findCommand(dir, command string, args []string) *exec.Cmd {
	makefile := filepath.Join(dir, "Makefile")
	if !FileExists(makefile) && !FileExists(filepath.Join(dir, "makefile")) {
		return nil
	}

	for _, variant := range GetCommandVariants(command) {
		if m.hasTarget(dir, variant) {
			cmdArgs := append([]string{variant}, args...)
			return exec.Command("make", cmdArgs...)
		}
	}
	return nil
}

func (m *makeRunner) hasTarget(dir, target string) bool {
	makefiles := []string{"Makefile", "makefile"}
	for _, mf := range makefiles {
		path := filepath.Join(dir, mf)
		if FileExists(path) {
			if found := m.checkTargetInFile(path, target); found {
				return true
			}
		}
	}
	return false
}

func (m *makeRunner) checkTargetInFile(path, target string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
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
	return false
}

type nodePackageRunner struct{}

func (n *nodePackageRunner) findCommand(dir, command string, args []string) *exec.Cmd {
	packageJSON := filepath.Join(dir, "package.json")
	if !FileExists(packageJSON) {
		return nil
	}

	data, err := os.ReadFile(packageJSON)
	if err != nil {
		return nil
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	// Check if script exists
	var scriptExists bool
	for _, variant := range GetCommandVariants(command) {
		if _, ok := pkg.Scripts[variant]; ok {
			command = variant
			scriptExists = true
			break
		}
	}

	// Special handling for typecheck in TypeScript projects
	if !scriptExists && command == "typecheck" {
		if FileExists(filepath.Join(dir, "tsconfig.json")) {
			// Use tsc for TypeScript type checking
			packageManager := detectPackageManager(dir)
			if packageManager == "" {
				return nil
			}
			// Try to run tsc directly if available
			cmdArgs := append([]string{"run", "tsc", "--noEmit"}, args...)
			return exec.Command(packageManager, cmdArgs...)
		}
	}

	if !scriptExists {
		return nil
	}

	// Determine which package manager to use
	packageManager := detectPackageManager(dir)
	if packageManager == "" {
		return nil
	}

	cmdArgs := append([]string{"run", command}, args...)
	return exec.Command(packageManager, cmdArgs...)
}

func detectPackageManager(dir string) string {
	// Priority order: bun > pnpm > yarn > npm > deno
	// Based on lockfiles and common usage patterns

	// Check for Bun
	if FileExists(filepath.Join(dir, "bun.lockb")) {
		return "bun"
	}

	// Check for pnpm
	if FileExists(filepath.Join(dir, "pnpm-lock.yaml")) {
		return "pnpm"
	}

	// Check for Yarn
	if FileExists(filepath.Join(dir, "yarn.lock")) {
		return "yarn"
	}

	// Check for npm
	if FileExists(filepath.Join(dir, "package-lock.json")) {
		return "npm"
	}

	// Check for Deno
	if FileExists(filepath.Join(dir, "deno.json")) || FileExists(filepath.Join(dir, "deno.jsonc")) {
		return "deno"
	}

	// Default to npm if package.json exists but no lockfile found
	if FileExists(filepath.Join(dir, "package.json")) {
		return "npm"
	}

	return ""
}

type cargoRunner struct{}

func (c *cargoRunner) findCommand(dir, command string, args []string) *exec.Cmd {
	cargoToml := filepath.Join(dir, "Cargo.toml")
	if !FileExists(cargoToml) {
		return nil
	}

	cargoCommands := map[string]string{
		"build":     "build",
		"run":       "run",
		"test":      "test",
		"lint":      "clippy",
		"format":    "fmt",
		"fmt":       "fmt",
		"clean":     "clean",
		"typecheck": "check",
	}

	for _, variant := range GetCommandVariants(command) {
		if cargoCmd, ok := cargoCommands[variant]; ok {
			cmdArgs := append([]string{cargoCmd}, args...)
			return exec.Command("cargo", cmdArgs...)
		}
	}
	return nil
}

type goRunner struct{}

func (g *goRunner) findCommand(dir, command string, args []string) *exec.Cmd {
	goMod := filepath.Join(dir, "go.mod")
	if !FileExists(goMod) {
		return nil
	}

	goCommands := map[string][]string{
		"build":     {"build"},
		"run":       {"run", "."},
		"test":      {"test", "./..."},
		"format":    {"fmt", "./..."},
		"fmt":       {"fmt", "./..."},
		"clean":     {"clean"},
		"install":   {"mod", "download"},
		"lint":      {"vet", "./..."},
		"typecheck": {"build", "-o", "/dev/null"},
	}

	for _, variant := range GetCommandVariants(command) {
		if goCmd, ok := goCommands[variant]; ok {
			cmdArgs := append(goCmd, args...)
			return exec.Command("go", cmdArgs...)
		}
	}
	return nil
}

type uvRunner struct{}

func (u *uvRunner) findCommand(dir, command string, args []string) *exec.Cmd {
	pyprojectToml := filepath.Join(dir, "pyproject.toml")
	if !FileExists(pyprojectToml) {
		return nil
	}

	data, err := os.ReadFile(pyprojectToml)
	if err != nil {
		return nil
	}

	content := string(data)
	hasUv := strings.Contains(content, "[tool.uv]") ||
		FileExists(filepath.Join(dir, "uv.lock"))

	if !hasUv {
		return nil
	}

	uvCommands := map[string][]string{
		"install":   {"sync"},
		"setup":     {"sync"},
		"run":       {"run"},
		"test":      {"run", "pytest"},
		"lint":      {"run", "ruff", "check"},
		"format":    {"run", "ruff", "format"},
		"fmt":       {"run", "ruff", "format"},
		"fix":       {"run", "ruff", "check", "--fix"},
		"typecheck": {"run", "pyright"},
	}

	for _, variant := range GetCommandVariants(command) {
		if uvCmd, ok := uvCommands[variant]; ok {
			cmdArgs := append(uvCmd, args...)
			return exec.Command("uv", cmdArgs...)
		}
	}
	return nil
}

type gradleRunner struct{}

func (g *gradleRunner) findCommand(dir, command string, args []string) *exec.Cmd {
	gradlew := filepath.Join(dir, "gradlew")
	buildGradle := filepath.Join(dir, "build.gradle")
	buildGradleKts := filepath.Join(dir, "build.gradle.kts")

	if !FileExists(buildGradle) && !FileExists(buildGradleKts) {
		return nil
	}

	gradleExec := "gradle"
	if FileExists(gradlew) {
		gradleExec = "./gradlew"
	}

	gradleCommands := map[string]string{
		"build": "build",
		"run":   "run",
		"test":  "test",
		"clean": "clean",
		"check": "check",
	}

	for _, variant := range GetCommandVariants(command) {
		if gradleCmd, ok := gradleCommands[variant]; ok {
			cmdArgs := append([]string{gradleCmd}, args...)
			return exec.Command(gradleExec, cmdArgs...)
		}
	}
	return nil
}

type mavenRunner struct{}

func (m *mavenRunner) findCommand(dir, command string, args []string) *exec.Cmd {
	pomXml := filepath.Join(dir, "pom.xml")
	if !FileExists(pomXml) {
		return nil
	}

	mvnw := filepath.Join(dir, "mvnw")
	mvnExec := "mvn"
	if FileExists(mvnw) {
		mvnExec = "./mvnw"
	}

	mavenCommands := map[string]string{
		"build":   "compile",
		"run":     "exec:java",
		"test":    "test",
		"clean":   "clean",
		"install": "install",
		"package": "package",
	}

	for _, variant := range GetCommandVariants(command) {
		if mvnCmd, ok := mavenCommands[variant]; ok {
			cmdArgs := append([]string{mvnCmd}, args...)
			return exec.Command(mvnExec, cmdArgs...)
		}
	}
	return nil
}
