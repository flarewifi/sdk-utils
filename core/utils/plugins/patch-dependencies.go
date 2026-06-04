package plugins

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// RequiredGoModule represents a module dependency with its path and version.
type RequiredGoModule struct {
	Path    string
	Version string
}

type ReplacedGoModule struct {
	Path       string
	Version    string
	NewPath    string
	NewVersion string
}

func PatchPluginDeps(pluginDir string) error {
	cmd := exec.Command("go", "mod", "tidy", "-e")
	cmd.Dir = pluginDir
	if err := cmd.Run(); err != nil {
		return err
	}

	systemDeps, err := GetInstalledModules()
	if err != nil {
		return err
	}

	pluginGoModFile := filepath.Join(pluginDir, "go.mod")
	if err := UpdateRequiredModules(pluginGoModFile, systemDeps); err != nil {
		return err
	}

	cmd = exec.Command("go", "mod", "tidy", "-e")
	cmd.Dir = pluginDir
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// GetRequiredGoModules parses the go.mod file and returns the dependencies with versions.
func GetRequiredGoModules(goModPath string) ([]RequiredGoModule, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()

	var dependencies []RequiredGoModule
	scanner := bufio.NewScanner(file)
	inRequireBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		} else if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		if inRequireBlock || strings.HasPrefix(line, "require ") {
			// Clean the line from "require" keyword if it's a single line require
			if strings.HasPrefix(line, "require ") {
				line = strings.TrimPrefix(line, "require ")
				line = strings.TrimSpace(line)
			}

			parts := strings.Fields(line)
			if len(parts) >= 2 {
				dependencies = append(dependencies, RequiredGoModule{
					Path:    parts[0],
					Version: parts[1],
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read go.mod: %w", err)
	}

	return dependencies, nil
}

func UpdateRequiredModules(goModPath string, replacements []RequiredGoModule) error {
	replacementMap := make(map[string]string)
	for _, r := range replacements {
		replacementMap[r.Path] = r.Version
	}

	file, err := os.Open(goModPath)
	if err != nil {
		return fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	inRequireBlock := false

	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "require (") {
			inRequireBlock = true
			lines = append(lines, scanner.Text())
			continue
		} else if inRequireBlock && strings.Contains(scanner.Text(), ")") {
			inRequireBlock = false
			lines = append(lines, scanner.Text())
			continue
		}

		inlineRequire := strings.HasPrefix(scanner.Text(), "require ")
		if inRequireBlock || inlineRequire {
			line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "require "))
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Check if the module is in the replacement list
				if newVersion, ok := replacementMap[parts[0]]; ok {
					indirect := strings.HasSuffix(scanner.Text(), "// indirect")
					requireLine := fmt.Sprintf("%s %s", parts[0], newVersion)
					if inlineRequire {
						// Append require prefix if present
						requireLine = "require " + requireLine
					} else {
						requireLine = "\t" + requireLine
					}

					if indirect {
						requireLine += " // indirect"
					}

					lines = append(lines, requireLine)
					continue
				}
			}

		}

		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read go.mod: %w", err)
	}

	if err := sdkutils.FsWriteFile(goModPath, []byte(strings.Join(lines, "\n"))); err != nil {
		return err
	}

	return nil
}

func GetInstalledModules() ([]RequiredGoModule, error) {
	coreMods, err := GetRequiredGoModules(filepath.Join(sdkutils.PathCoreDir, "go.mod"))
	if err != nil {
		return nil, err
	}

	utilsMods, err := GetRequiredGoModules(filepath.Join(sdkutils.PathSdkDir, "utils", "go.mod"))
	if err != nil {
		return nil, err
	}

	apiMods, err := GetRequiredGoModules(filepath.Join(sdkutils.PathSdkDir, "api", "go.mod"))
	if err != nil {
		return nil, err
	}

	pluginModules := []RequiredGoModule{}

	installedDirs := InstalledPluginDirs()
	for _, pluginDir := range installedDirs {
		pluginGoModFile := filepath.Join(pluginDir, "go.mod")

		if !sdkutils.FsExists(pluginGoModFile) {
			os.RemoveAll(pluginDir)
			continue
		}

		reqMods, err := GetRequiredGoModules(pluginGoModFile)
		if err != nil {
			return nil, err
		}
		pluginModules = append(pluginModules, reqMods...)
	}

	modules := []RequiredGoModule{}
	modGroups := [][]RequiredGoModule{coreMods, utilsMods, apiMods, pluginModules}
	for _, mods := range modGroups {
		for _, mod := range mods {
			// Check if the module is already in the list
			found := false
			for _, m := range modules {
				if m.Path == mod.Path {
					found = true
					break
				}
			}
			if !found {
				modules = append(modules, mod)
			}
		}
	}

	return modules, nil
}
