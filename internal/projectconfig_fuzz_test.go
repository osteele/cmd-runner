package internal

import (
	"encoding/json"
	"strings"
	"testing"
)

func FuzzStandardizeJSONC(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"tasks":{"test":"deno test"}}`),
		[]byte("{\n// comment\n\"tasks\": {\"test\": \"deno test\",},\n}\n"),
		[]byte(`{"url":"https://example.com/*path*/"}`),
		[]byte(`{/* unterminated`),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		standardJSON, err := standardizeJSONC(input)
		if err != nil {
			return
		}
		if len(standardJSON) > len(input) {
			t.Fatalf("standardized JSON grew from %d to %d bytes", len(input), len(standardJSON))
		}
		if json.Valid(input) && !json.Valid(standardJSON) {
			t.Fatalf("valid JSON became invalid: input=%q output=%q", input, standardJSON)
		}
	})
}

func FuzzPythonProjectConfig(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("[project]\ndependencies = [\"mypy>=1\"]\n"),
		[]byte("[tool.poetry.group.dev.dependencies]\npyright = \"^1.1\"\n"),
		[]byte("[tool.uv]\ndev-dependencies = [\"ruff\"]\n"),
		[]byte("[project"),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		config, err := parsePythonProjectConfig(input)
		if err != nil {
			return
		}
		for _, name := range []string{"uv", "poetry", "ruff", "mypy", "pyright"} {
			_ = config.hasTool(name)
			_ = config.hasDependency(name)
		}
	})
}

func FuzzCargoManifest(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("[package]\nname = \"fixture\"\nversion = \"0.1.0\"\n"),
		[]byte("[package]\nautobins = false\n\n[[bin]]\nname = \"worker\"\n"),
		[]byte("[[bin"),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		manifest, err := parseCargoManifest(input)
		if err != nil {
			return
		}
		for _, binary := range manifest.Binaries {
			_ = binary.Name
		}
	})
}

func FuzzNodePackage(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"scripts":{"test":"vitest"},"devDependencies":{"eslint":"latest"}}`),
		[]byte(`{"dependencies":{"typescript":"5"}}`),
		[]byte(`{"scripts":`),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		packageConfig, err := parseNodePackage(input)
		if err != nil {
			return
		}
		for _, name := range []string{"eslint", "typescript", "ruff"} {
			_ = packageConfig.hasDependency(name)
		}
	})
}

func FuzzDependencyName(f *testing.F) {
	for _, seed := range []string{"mypy>=1.15", "PyRight[all]", "ruff_lsp", "", "éclair"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, specification string) {
		name := dependencyName(specification)
		if name != strings.ToLower(name) {
			t.Fatalf("dependency name is not normalized: %q", name)
		}
	})
}
