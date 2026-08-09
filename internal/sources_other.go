package internal

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// CargoSource for Rust projects
type CargoSource struct {
	baseSource
}

func NewCargoSource(dir string) CommandSource {
	if !FileExists(filepath.Join(dir, "Cargo.toml")) {
		return nil
	}

	return &CargoSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "Cargo",
			priority: 10,
		},
	}
}

func (c *CargoSource) ListCommands() map[string]CommandInfo {
	commands := map[string]CommandInfo{
		"build":   {Description: "Build the project", Execution: "cargo build"},
		"run":     {Description: "Run the project", Execution: "cargo run"},
		"test":    {Description: "Run tests", Execution: "cargo test"},
		"check":   {Description: "Check code for errors", Execution: "cargo check"},
		"format":  {Description: "Format code", Execution: "cargo fmt"},
		"lint":    {Description: "Run clippy linter", Execution: "cargo clippy"},
		"clean":   {Description: "Clean build artifacts", Execution: "cargo clean"},
		"setup":   {Description: "Download dependencies", Execution: "cargo fetch"},
		"install": {Description: "Install binary globally", Execution: "cargo install --path ."},
	}
	for _, binaryName := range cargoBinaryNames(c.dir) {
		commands["run:"+binaryName] = CommandInfo{
			Description: "Run the " + binaryName + " binary",
			Execution:   "cargo run --bin " + binaryName,
		}
	}
	return commands
}

func (c *CargoSource) FindCommand(command string, args []string) *exec.Cmd {
	cargoCommands := map[string]string{
		"build":     "build",
		"run":       "run",
		"test":      "test",
		"lint":      "clippy",
		"format":    "fmt",
		"clean":     "clean",
		"typecheck": "check",
		"check":     "check",
		"fix":       "fix",
		"setup":     "fetch",
		"install":   "install",
		"publish":   "publish",
	}

	if cargoCmd, ok := cargoCommands[command]; ok {
		var cmdArgs []string
		if cargoCmd == "install" {
			// Modern cargo requires --path for installing from current directory
			cmdArgs = append([]string{"install", "--path", "."}, args...)
		} else {
			cmdArgs = append([]string{cargoCmd}, args...)
		}
		cmd := exec.Command("cargo", cmdArgs...)
		cmd.Dir = c.dir
		return cmd
	}

	// Handle declared binary targets (run:binary-name pattern).
	if strings.HasPrefix(command, "run:") {
		binaryName := strings.TrimPrefix(command, "run:")
		for _, knownBinary := range cargoBinaryNames(c.dir) {
			if knownBinary == binaryName {
				cmdArgs := append([]string{"run", "--bin", binaryName}, args...)
				cmd := exec.Command("cargo", cmdArgs...)
				cmd.Dir = c.dir
				return cmd
			}
		}
	}

	return nil
}

// GoSource for Go projects
type GoSource struct {
	baseSource
	discoveryCache  goDiscoveryCache
	discoverProject func(string) goProjectDiscovery
}

func NewGoSource(dir string) CommandSource {
	if !FileExists(filepath.Join(dir, "go.mod")) && !FileExists(filepath.Join(dir, "go.work")) {
		return nil
	}

	return &GoSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "Go",
			priority: 10,
		},
		discoverProject: discoverGoProject,
	}
}

func (g *GoSource) ListCommands() map[string]CommandInfo {
	discovery := g.discovery()
	patterns := strings.Join(discovery.packagePatterns, " ")
	commands := map[string]CommandInfo{
		"build":  {Description: "Build the project", Execution: "go build " + patterns},
		"test":   {Description: "Run tests", Execution: "go test " + patterns},
		"format": {Description: "Format code", Execution: "go fmt " + patterns},
		"lint":   {Description: "Run linter", Execution: "go vet " + patterns},
		"clean":  {Description: "Clean build artifacts", Execution: "go clean"},
	}
	if discovery.workspace {
		commands["setup"] = CommandInfo{Description: "Synchronize workspace dependencies", Execution: "go work sync"}
	} else {
		commands["setup"] = CommandInfo{Description: "Download dependencies", Execution: "go mod download"}
	}
	mainPackages := discovery.mainPackages
	if len(mainPackages) > 0 {
		commands["install"] = CommandInfo{
			Description: "Install binaries globally",
			Execution:   "go install " + strings.Join(mainPackages, " "),
		}
	}
	if len(mainPackages) == 1 {
		commands["run"] = CommandInfo{
			Description: "Run the project",
			Execution:   "go run " + mainPackages[0],
		}
	} else if len(mainPackages) > 1 {
		for commandName, target := range goRunTargets(mainPackages) {
			commands["run:"+commandName] = CommandInfo{
				Description: "Run " + target,
				Execution:   "go run " + target,
			}
		}
	}
	return commands
}

func (g *GoSource) FindCommand(command string, args []string) *exec.Cmd {
	discovery := g.discovery()
	goCommands := map[string][]string{
		"build":     append([]string{"build"}, discovery.packagePatterns...),
		"test":      append([]string{"test"}, discovery.packagePatterns...),
		"format":    append([]string{"fmt"}, discovery.packagePatterns...),
		"clean":     {"clean"},
		"lint":      append([]string{"vet"}, discovery.packagePatterns...),
		"typecheck": append([]string{"build"}, discovery.packagePatterns...),
	}
	if discovery.workspace {
		goCommands["setup"] = []string{"work", "sync"}
	} else {
		goCommands["setup"] = []string{"mod", "download"}
	}

	if goCmd, ok := goCommands[command]; ok {
		return g.command(append(goCmd, args...))
	}

	mainPackages := discovery.mainPackages
	if command == "run" && len(mainPackages) == 1 {
		return g.command(append([]string{"run", mainPackages[0]}, args...))
	}
	if command == "install" && len(mainPackages) > 0 {
		cmdArgs := append([]string{"install"}, mainPackages...)
		return g.command(append(cmdArgs, args...))
	}
	if strings.HasPrefix(command, "run:") {
		if target, exists := goRunTargets(mainPackages)[strings.TrimPrefix(command, "run:")]; exists {
			return g.command(append([]string{"run", target}, args...))
		}
	}

	return nil
}

func (g *GoSource) command(args []string) *exec.Cmd {
	cmd := exec.Command("go", args...)
	cmd.Dir = g.dir
	return cmd
}

func goRunTargets(mainPackages []string) map[string]string {
	baseNameCounts := make(map[string]int)
	for _, target := range mainPackages {
		baseNameCounts[filepath.Base(target)]++
	}
	targets := make(map[string]string, len(mainPackages))
	for _, target := range mainPackages {
		name := filepath.Base(target)
		if baseNameCounts[name] > 1 {
			name = strings.TrimPrefix(target, "./")
		}
		targets[name] = target
	}
	return targets
}

// GradleSource for Gradle projects
type GradleSource struct {
	baseSource
}

func NewGradleSource(dir string) CommandSource {
	if !FileExists(filepath.Join(dir, "build.gradle")) &&
		!FileExists(filepath.Join(dir, "build.gradle.kts")) {
		return nil
	}

	return &GradleSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "Gradle",
			priority: 10,
		},
	}
}

func (g *GradleSource) ListCommands() map[string]CommandInfo {
	gradleExec := "gradle"
	if FileExists(filepath.Join(g.dir, "gradlew")) {
		gradleExec = "./gradlew"
	}
	return map[string]CommandInfo{
		"build":   {Description: "Build the project", Execution: gradleExec + " build"},
		"run":     {Description: "Run the project", Execution: gradleExec + " run"},
		"test":    {Description: "Run tests", Execution: gradleExec + " test"},
		"clean":   {Description: "Clean build artifacts", Execution: gradleExec + " clean"},
		"check":   {Description: "Run checks", Execution: gradleExec + " check"},
		"setup":   {Description: "Download dependencies", Execution: gradleExec + " build"},
		"install": {Description: "Install application (requires application plugin)", Execution: gradleExec + " installDist"},
	}
}

func (g *GradleSource) FindCommand(command string, args []string) *exec.Cmd {
	gradleExec := "gradle"
	if FileExists(filepath.Join(g.dir, "gradlew")) {
		gradleExec = "./gradlew"
	}

	gradleCommands := map[string]string{
		"build":   "build",
		"run":     "run",
		"test":    "test",
		"clean":   "clean",
		"check":   "check",
		"setup":   "build",
		"install": "installDist",
	}

	if gradleCmd, ok := gradleCommands[command]; ok {
		var cmdArgs []string
		// Handle commands with multiple parts (like "dependencies --write-locks")
		if strings.Contains(gradleCmd, " ") {
			parts := strings.Fields(gradleCmd)
			cmdArgs = append(parts, args...)
		} else {
			cmdArgs = append([]string{gradleCmd}, args...)
		}
		cmd := exec.Command(gradleExec, cmdArgs...)
		cmd.Dir = g.dir
		return cmd
	}

	return nil
}

// MavenSource for Maven projects
type MavenSource struct {
	baseSource
}

func NewMavenSource(dir string) CommandSource {
	if !FileExists(filepath.Join(dir, "pom.xml")) {
		return nil
	}

	return &MavenSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "Maven",
			priority: 10,
		},
	}
}

func (m *MavenSource) ListCommands() map[string]CommandInfo {
	mvnExec := "mvn"
	if FileExists(filepath.Join(m.dir, "mvnw")) {
		mvnExec = "./mvnw"
	}
	return map[string]CommandInfo{
		"build":   {Description: "Build the project", Execution: mvnExec + " compile"},
		"run":     {Description: "Run the project", Execution: mvnExec + " exec:java"},
		"test":    {Description: "Run tests", Execution: mvnExec + " test"},
		"clean":   {Description: "Clean build artifacts", Execution: mvnExec + " clean"},
		"setup":   {Description: "Download dependencies", Execution: mvnExec + " dependency:resolve"},
		"install": {Description: "Install to local Maven repository", Execution: mvnExec + " install"},
		"package": {Description: "Package the project", Execution: mvnExec + " package"},
	}
}

func (m *MavenSource) FindCommand(command string, args []string) *exec.Cmd {
	mvnExec := "mvn"
	if FileExists(filepath.Join(m.dir, "mvnw")) {
		mvnExec = "./mvnw"
	}

	mavenCommands := map[string]string{
		"build":   "compile",
		"run":     "exec:java",
		"test":    "test",
		"clean":   "clean",
		"setup":   "dependency:resolve",
		"install": "install",
		"package": "package",
	}

	if mvnCmd, ok := mavenCommands[command]; ok {
		cmdArgs := append([]string{mvnCmd}, args...)
		cmd := exec.Command(mvnExec, cmdArgs...)
		cmd.Dir = m.dir
		return cmd
	}

	return nil
}
