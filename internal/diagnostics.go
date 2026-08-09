package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func (project *Project) configurationDiagnostics() []string {
	dir := project.Dir
	diagnostics := make([]string, 0)
	addError := func(err error) {
		if err != nil {
			diagnostics = append(diagnostics, err.Error())
		}
	}

	if FileExists(filepath.Join(dir, ".mise.toml")) {
		var config map[string]any
		addError(decodeTOMLFile(filepath.Join(dir, ".mise.toml"), &config))
	}
	if FileExists(filepath.Join(dir, "package.json")) {
		_, err := readNodePackage(dir)
		addError(err)
	}
	for _, filename := range []string{"deno.json", "deno.jsonc"} {
		path := filepath.Join(dir, filename)
		if !FileExists(path) {
			continue
		}
		var config struct {
			Tasks map[string]string `json:"tasks"`
		}
		addError(decodeJSONCFile(path, &config))
	}
	if FileExists(filepath.Join(dir, "pyproject.toml")) {
		_, err := readPythonProjectConfig(dir)
		addError(err)
	}
	if FileExists(filepath.Join(dir, "Cargo.toml")) {
		_, err := readCargoManifest(dir)
		addError(err)
	}
	if FileExists(filepath.Join(dir, "go.work")) {
		addError(validateGoConfiguration(dir, "work", filepath.Join(dir, "go.work")))
	} else if FileExists(filepath.Join(dir, "go.mod")) {
		addError(validateGoConfiguration(dir, "mod", filepath.Join(dir, "go.mod")))
	}

	sort.Strings(diagnostics)
	return diagnostics
}

func validateGoConfiguration(dir, kind, path string) error {
	command := exec.Command("go", kind, "edit", "-json")
	command.Dir = dir
	if kind == "mod" {
		command.Env = append(os.Environ(), "GOWORK=off")
	}
	if output, err := command.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("parse %s: %s", path, message)
	}
	return nil
}
