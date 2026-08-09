package internal

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type goProjectDiscovery struct {
	mainPackages    []string
	packagePatterns []string
	workspace       bool
}

type goDiscoveryCache struct {
	sync.Mutex
	fingerprint string
	discovery   goProjectDiscovery
}

func (g *GoSource) discovery() goProjectDiscovery {
	fingerprint := goSourceFingerprint(g.dir)
	g.discoveryCache.Lock()
	if g.discoveryCache.fingerprint == fingerprint {
		discovery := cloneGoDiscovery(g.discoveryCache.discovery)
		g.discoveryCache.Unlock()
		return discovery
	}
	g.discoveryCache.Unlock()

	discoverProject := g.discoverProject
	if discoverProject == nil {
		discoverProject = discoverGoProject
	}
	discovery := discoverProject(g.dir)
	g.discoveryCache.Lock()
	g.discoveryCache.fingerprint = fingerprint
	g.discoveryCache.discovery = cloneGoDiscovery(discovery)
	g.discoveryCache.Unlock()
	return discovery
}

func cloneGoDiscovery(discovery goProjectDiscovery) goProjectDiscovery {
	discovery.mainPackages = append([]string(nil), discovery.mainPackages...)
	discovery.packagePatterns = append([]string(nil), discovery.packagePatterns...)
	return discovery
}

func discoverGoProject(dir string) goProjectDiscovery {
	moduleRoots, workspace := goModuleRoots(dir)
	patterns := make([]string, 0, len(moduleRoots))
	mainPackages := make(map[string]bool)
	for _, moduleRoot := range moduleRoots {
		pattern := goPackagePattern(dir, moduleRoot)
		patterns = append(patterns, pattern)
		for _, mainPackage := range listGoMainPackages(dir, moduleRoot) {
			mainPackages[mainPackage] = true
		}
	}
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	return goProjectDiscovery{
		mainPackages:    sortedKeys(mainPackages),
		packagePatterns: patterns,
		workspace:       workspace,
	}
}

func goModuleRoots(dir string) ([]string, bool) {
	if FileExists(filepath.Join(dir, "go.work")) {
		workspaceCommand := exec.Command("go", "work", "edit", "-json")
		workspaceCommand.Dir = dir
		if output, err := workspaceCommand.Output(); err == nil {
			var workspace struct {
				Use []struct {
					DiskPath string
				}
			}
			if json.Unmarshal(output, &workspace) == nil {
				roots := make([]string, 0, len(workspace.Use))
				for _, use := range workspace.Use {
					root := use.DiskPath
					if !filepath.IsAbs(root) {
						root = filepath.Join(dir, root)
					}
					if pathWithin(dir, root) && FileExists(filepath.Join(root, "go.mod")) {
						roots = append(roots, filepath.Clean(root))
					}
				}
				sort.Strings(roots)
				return roots, true
			}
		}
	}
	if FileExists(filepath.Join(dir, "go.mod")) {
		return []string{dir}, false
	}
	return nil, false
}

func pathWithin(root, path string) bool {
	relativePath, err := filepath.Rel(root, path)
	return err == nil && relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator))
}

func goPackagePattern(root, moduleRoot string) string {
	relativePath, err := filepath.Rel(root, moduleRoot)
	if err != nil || relativePath == "." {
		return "./..."
	}
	return "./" + filepath.ToSlash(relativePath) + "/..."
}

func listGoMainPackages(root, moduleRoot string) []string {
	listCommand := exec.Command("go", "list", "-f", `{{if eq .Name "main"}}{{.Dir}}{{end}}`, "./...")
	listCommand.Dir = moduleRoot
	output, err := listCommand.Output()
	if err != nil {
		return discoverMainPackagesFromFiles(root, moduleRoot)
	}

	packages := make([]string, 0)
	for _, directory := range strings.Split(string(output), "\n") {
		directory = strings.TrimSpace(directory)
		if directory == "" {
			continue
		}
		if target, ok := goPackageTarget(root, directory); ok {
			packages = append(packages, target)
		}
	}
	sort.Strings(packages)
	return packages
}

func goPackageTarget(root, directory string) (string, bool) {
	relativeDirectory, err := filepath.Rel(root, directory)
	if err != nil || relativeDirectory == ".." || strings.HasPrefix(relativeDirectory, ".."+string(filepath.Separator)) {
		return "", false
	}
	if relativeDirectory == "." {
		return ".", true
	}
	return "./" + filepath.ToSlash(relativeDirectory), true
}

func discoverMainPackagesFromFiles(root, moduleRoot string) []string {
	candidates := []string{moduleRoot}
	commandDirectories, _ := filepath.Glob(filepath.Join(moduleRoot, "cmd", "*"))
	candidates = append(candidates, commandDirectories...)

	packages := make([]string, 0)
	for _, candidate := range candidates {
		if !directoryContainsMainPackage(candidate) {
			continue
		}
		if target, ok := goPackageTarget(root, candidate); ok {
			packages = append(packages, target)
		}
	}
	sort.Strings(packages)
	return packages
}

func directoryContainsMainPackage(dir string) bool {
	files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	for _, filename := range files {
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}
		parsedFile, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.PackageClauseOnly)
		if err == nil && parsedFile.Name.Name == "main" {
			return true
		}
	}
	return false
}

func goSourceFingerprint(root string) string {
	hash := sha256.New()
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(hash, "error:%s:%v\x00", path, err)
			return nil
		}
		if entry.IsDir() && path != root && shouldSkipGoDiscoveryDirectory(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.IsDir() || !isGoDiscoveryFile(entry.Name()) {
			return nil
		}
		relativePath, _ := filepath.Rel(root, path)
		fmt.Fprintf(hash, "%s\x00", filepath.ToSlash(relativePath))
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			fmt.Fprintf(hash, "error:%v\x00", readErr)
			return nil
		}
		_, _ = hash.Write(data)
		_, _ = hash.Write([]byte{0})
		return nil
	})
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func shouldSkipGoDiscoveryDirectory(name string) bool {
	return name == ".git" || name == ".jj" || name == "vendor" || name == "node_modules"
}

func isGoDiscoveryFile(name string) bool {
	return strings.HasSuffix(name, ".go") || name == "go.mod" || name == "go.sum" ||
		name == "go.work" || name == "go.work.sum"
}
