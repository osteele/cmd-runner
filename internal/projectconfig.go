package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type pythonProjectConfig struct {
	Project struct {
		Dependencies         []string            `toml:"dependencies"`
		OptionalDependencies map[string][]string `toml:"optional-dependencies"`
	} `toml:"project"`
	DependencyGroups map[string][]any `toml:"dependency-groups"`
	Tool             map[string]any   `toml:"tool"`
}

func readPythonProjectConfig(dir string) (*pythonProjectConfig, error) {
	var config pythonProjectConfig
	if err := decodeTOMLFile(filepath.Join(dir, "pyproject.toml"), &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (config *pythonProjectConfig) hasTool(name string) bool {
	_, exists := config.Tool[name]
	return exists
}

func (config *pythonProjectConfig) hasDependency(name string) bool {
	for _, dependency := range config.Project.Dependencies {
		if dependencyName(dependency) == name {
			return true
		}
	}
	for _, dependencies := range config.Project.OptionalDependencies {
		for _, dependency := range dependencies {
			if dependencyName(dependency) == name {
				return true
			}
		}
	}
	if containsDependency(config.DependencyGroups, name) {
		return true
	}
	return poetryHasDependency(config.Tool["poetry"], name) ||
		uvHasDependency(config.Tool["uv"], name)
}

func poetryHasDependency(value any, name string) bool {
	poetry, ok := value.(map[string]any)
	if !ok {
		return false
	}
	if dependencyMapContains(poetry["dependencies"], name) ||
		dependencyMapContains(poetry["dev-dependencies"], name) {
		return true
	}
	groups, ok := poetry["group"].(map[string]any)
	if !ok {
		return false
	}
	for _, groupValue := range groups {
		group, ok := groupValue.(map[string]any)
		if ok && dependencyMapContains(group["dependencies"], name) {
			return true
		}
	}
	return false
}

func uvHasDependency(value any, name string) bool {
	uv, ok := value.(map[string]any)
	return ok && containsDependency(uv["dev-dependencies"], name)
}

func dependencyMapContains(value any, name string) bool {
	dependencies, ok := value.(map[string]any)
	if !ok {
		return false
	}
	for dependency := range dependencies {
		if dependencyName(dependency) == name {
			return true
		}
	}
	return false
}

func containsDependency(value any, name string) bool {
	switch value := value.(type) {
	case string:
		return dependencyName(value) == name
	case []any:
		for _, item := range value {
			if containsDependency(item, name) {
				return true
			}
		}
	case map[string]any:
		for key, item := range value {
			if dependencyName(key) == name || containsDependency(item, name) {
				return true
			}
		}
	case map[string][]any:
		for _, items := range value {
			if containsDependency(items, name) {
				return true
			}
		}
	}
	return false
}

func dependencyName(specification string) string {
	specification = strings.TrimSpace(specification)
	end := 0
	for end < len(specification) {
		character := specification[end]
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' || character == '.' {
			end++
			continue
		}
		break
	}
	return strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(specification[:end]))
}

type cargoManifest struct {
	Binaries []struct {
		Name string `toml:"name"`
	} `toml:"bin"`
}

func readCargoManifest(dir string) (*cargoManifest, error) {
	var manifest cargoManifest
	if err := decodeTOMLFile(filepath.Join(dir, "Cargo.toml"), &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func decodeTOMLFile(path string, destination any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := toml.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

type nodePackage struct {
	Scripts         map[string]string          `json:"scripts"`
	Dependencies    map[string]json.RawMessage `json:"dependencies"`
	DevDependencies map[string]json.RawMessage `json:"devDependencies"`
}

func readNodePackage(dir string) (*nodePackage, error) {
	var packageConfig nodePackage
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &packageConfig); err != nil {
		return nil, err
	}
	return &packageConfig, nil
}

func (packageConfig *nodePackage) hasDependency(name string) bool {
	if _, exists := packageConfig.Dependencies[name]; exists {
		return true
	}
	_, exists := packageConfig.DevDependencies[name]
	return exists
}

func decodeJSONCFile(path string, destination any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	standardJSON, err := standardizeJSONC(data)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if err := json.Unmarshal(standardJSON, destination); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func standardizeJSONC(data []byte) ([]byte, error) {
	withoutComments := append([]byte(nil), data...)
	inString := false
	escaped := false
	for index := 0; index < len(withoutComments); index++ {
		character := withoutComments[index]
		if inString {
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == '"' {
				inString = false
			}
			continue
		}
		if character == '"' {
			inString = true
			continue
		}
		if character != '/' || index+1 >= len(withoutComments) {
			continue
		}

		switch withoutComments[index+1] {
		case '/':
			withoutComments[index], withoutComments[index+1] = ' ', ' '
			index += 2
			for index < len(withoutComments) && withoutComments[index] != '\n' {
				withoutComments[index] = ' '
				index++
			}
			index--
		case '*':
			withoutComments[index], withoutComments[index+1] = ' ', ' '
			index += 2
			closed := false
			for index < len(withoutComments)-1 {
				if withoutComments[index] == '*' && withoutComments[index+1] == '/' {
					withoutComments[index], withoutComments[index+1] = ' ', ' '
					index++
					closed = true
					break
				}
				if withoutComments[index] != '\n' && withoutComments[index] != '\r' {
					withoutComments[index] = ' '
				}
				index++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated block comment")
			}
		}
	}
	if inString {
		return nil, fmt.Errorf("unterminated string")
	}

	result := bytes.NewBuffer(make([]byte, 0, len(withoutComments)))
	inString = false
	escaped = false
	for index, character := range withoutComments {
		if inString {
			result.WriteByte(character)
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == '"' {
				inString = false
			}
			continue
		}
		if character == '"' {
			inString = true
			result.WriteByte(character)
			continue
		}
		if character == ',' {
			next := index + 1
			for next < len(withoutComments) && isJSONWhitespace(withoutComments[next]) {
				next++
			}
			if next < len(withoutComments) && (withoutComments[next] == '}' || withoutComments[next] == ']') {
				continue
			}
		}
		result.WriteByte(character)
	}
	return result.Bytes(), nil
}

func isJSONWhitespace(character byte) bool {
	return character == ' ' || character == '\t' || character == '\r' || character == '\n'
}
