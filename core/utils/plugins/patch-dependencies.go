package plugins

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarewifi/sdk-utils"
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

// PatchPluginDeps aligns a plugin's dependency versions with the host before it is
// compiled, so its .so is ABI-compatible with the core and the other plugins it
// loads alongside.
//
// pinned is the authoritative per-core-version dependency lock supplied by the
// build server (empty for local dev builds and for the first plugin built against a
// core version). When present, its versions are forced exactly via `replace`
// (overriding the locally-derived versions and surviving MVS), and the resulting
// go.sum is verified against the locked content hashes. When empty, behavior is
// unchanged: versions are aligned to the locally-installed core/sdk/plugin set on a
// best-effort require basis.
func PatchPluginDeps(pluginDir string, pinned []LockedGoModule) error {
	cmd := exec.Command("go", "mod", "tidy", "-e")
	cmd.Dir = pluginDir
	if err := cmd.Run(); err != nil {
		return err
	}

	systemDeps, err := GetInstalledModules()
	if err != nil {
		return err
	}

	// The lock takes precedence over the locally-derived versions: it is the set
	// every other plugin for this core version was already pinned to.
	replacements := systemDeps
	for _, p := range pinned {
		replacements = append(replacements, RequiredGoModule{Path: p.Path, Version: p.Version})
	}

	pluginGoModFile := filepath.Join(pluginDir, "go.mod")
	if err := UpdateRequiredModules(pluginGoModFile, replacements); err != nil {
		return err
	}

	// Force the locked versions exactly (require alone can be overridden by MVS).
	if err := forcePinnedVersions(pluginDir, pinned); err != nil {
		return err
	}

	cmd = exec.Command("go", "mod", "tidy", "-e")
	cmd.Dir = pluginDir
	if err := cmd.Run(); err != nil {
		return err
	}

	// Confirm the pinned modules resolved to the locked content, not just the
	// locked label, before we trust the resulting .so.
	if err := VerifyPinnedGoSum(pluginDir, pinned); err != nil {
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

	// Collect every dependency together with the source that declared it, so we
	// can report conflicts. Order matters: the first source to declare a module
	// wins (core > sdk/utils > sdk/api > installed plugins). Because plugin .so
	// files must be built against the exact same versions as the host, any
	// divergence here would otherwise be silently overridden and only surface as
	// a cryptic "plugin was built with a different version of package" error at
	// plugin.Open() time. We surface it loudly at build time instead.
	type sourcedModule struct {
		mod    RequiredGoModule
		source string
	}

	var all []sourcedModule
	addGroup := func(mods []RequiredGoModule, source string) {
		for _, m := range mods {
			all = append(all, sourcedModule{mod: m, source: source})
		}
	}
	addGroup(coreMods, "core")
	addGroup(utilsMods, "sdk/utils")
	addGroup(apiMods, "sdk/api")

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
		addGroup(reqMods, "plugin "+filepath.Base(pluginDir))
	}

	chosen := make(map[string]sourcedModule)
	modules := []RequiredGoModule{}
	for _, sm := range all {
		prev, ok := chosen[sm.mod.Path]
		if !ok {
			chosen[sm.mod.Path] = sm
			modules = append(modules, sm.mod)
			continue
		}
		// Already pinned by a higher-priority source. Warn if this source wanted
		// a different version, since it will be overridden to keep ABI parity.
		if prev.mod.Version != sm.mod.Version {
			fmt.Printf(
				"Warning: dependency version conflict for %s: pinning %s (from %s), overriding %s required by %s.\n",
				sm.mod.Path, prev.mod.Version, prev.source, sm.mod.Version, sm.source,
			)
		}
	}

	return modules, nil
}
