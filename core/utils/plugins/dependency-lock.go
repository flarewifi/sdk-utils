package plugins

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LockedGoModule is one entry of a per-core-version dependency lock: a module
// pinned to an exact version AND its go.sum content hashes. The hashes matter
// because path+version is only a label — a moved tag, a `replace`, or proxy
// divergence can change the bytes, and a Go plugin's ABI check rejects a .so whose
// shared packages differ in content. The JSON tags are the wire contract with the
// builder, which shuttles these between the server and the flare CLI as a file.
type LockedGoModule struct {
	Path      string `json:"path"`
	Version   string `json:"version"`
	Hash      string `json:"hash"`        // go.sum "h1:" for `path version`
	GoModHash string `json:"go_mod_hash"` // go.sum "h1:" for `path version/go.mod`
}

type goSumHashes struct {
	zip   string
	goMod string
}

// ResolvedGoModules reports the external module set a freshly-built plugin was
// compiled against: the selected versions from its (tidied) go.mod require block,
// each annotated with its go.sum hashes. Workspace-local modules (sdk/api,
// sdk/utils, core — supplied via go.work, never downloaded) carry no go.sum entry
// and are therefore naturally excluded; only versioned, content-hashed external
// modules end up in the lock.
func ResolvedGoModules(pluginDir string) ([]LockedGoModule, error) {
	goModPath := filepath.Join(pluginDir, "go.mod")
	required, err := GetRequiredGoModules(goModPath)
	if err != nil {
		return nil, err
	}

	// A module-version `replace` (e.g. the pins forcePinnedVersions writes) changes
	// the version that is actually downloaded and linked, while the require line can
	// still read the higher MVS-selected version. Resolve to the effective version
	// so go.sum keys match and the version recorded is the one truly built.
	replaces, err := parseModuleReplaces(goModPath)
	if err != nil {
		return nil, err
	}

	sums, err := readGoSum(filepath.Join(pluginDir, "go.sum"))
	if err != nil {
		return nil, err
	}

	resolved := make([]LockedGoModule, 0, len(required))
	for _, m := range required {
		lookupPath, version := m.Path, m.Version
		if r, ok := replaces[m.Path]; ok {
			lookupPath, version = r.Path, r.Version
		}
		h, ok := sums[lookupPath+" "+version]
		if !ok {
			// No go.sum entry => workspace-local/filesystem-replaced module, not a
			// downloaded dependency. It is identical for every plugin built against
			// this core commit, so it does not belong in the lock.
			continue
		}
		resolved = append(resolved, LockedGoModule{
			Path:      m.Path,
			Version:   version,
			Hash:      h.zip,
			GoModHash: h.goMod,
		})
	}
	return resolved, nil
}

// VerifyPinnedGoSum fails if any pinned module resolved to different bytes than the
// lock records — the moved-tag / `replace` / proxy-divergence hazard that pinning
// the version alone cannot catch. Modules the plugin does not actually use (absent
// from its go.sum) are skipped.
func VerifyPinnedGoSum(pluginDir string, pinned []LockedGoModule) error {
	sums, err := readGoSum(filepath.Join(pluginDir, "go.sum"))
	if err != nil {
		return err
	}

	for _, p := range pinned {
		h, ok := sums[p.Path+" "+p.Version]
		if !ok {
			continue
		}
		if p.Hash != "" && h.zip != "" && p.Hash != h.zip {
			return fmt.Errorf(
				"dependency lock hash mismatch for %s %s: locked %s but resolved %s — the version label maps to different content (moved tag or replace)",
				p.Path, p.Version, p.Hash, h.zip)
		}
		if p.GoModHash != "" && h.goMod != "" && p.GoModHash != h.goMod {
			return fmt.Errorf(
				"dependency lock go.mod hash mismatch for %s %s: locked %s but resolved %s",
				p.Path, p.Version, p.GoModHash, h.goMod)
		}
	}
	return nil
}

// forcePinnedVersions rewrites the plugin's go.mod so the pinned modules resolve to
// EXACTLY the locked versions. A bare `require` rewrite is not enough: Go's minimal
// version selection bumps a dependency back up when another module requires a higher
// version, so a pin that lowers a version would silently not hold (and then loop
// against the server's record-time conflict check). A `replace path => path@version`
// forces the exact version regardless of MVS. Only modules already in the plugin's
// build graph are replaced, so unused lock entries don't litter the shipped go.mod.
func forcePinnedVersions(pluginDir string, pinned []LockedGoModule) error {
	if len(pinned) == 0 {
		return nil
	}

	required, err := GetRequiredGoModules(filepath.Join(pluginDir, "go.mod"))
	if err != nil {
		return err
	}
	inGraph := make(map[string]bool, len(required))
	for _, m := range required {
		inGraph[m.Path] = true
	}

	for _, p := range pinned {
		if p.Version == "" || !inGraph[p.Path] {
			continue
		}
		cmd := exec.Command("go", "mod", "edit", "-replace="+p.Path+"="+p.Path+"@"+p.Version)
		cmd.Dir = pluginDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("go mod edit -replace %s@%s: %w: %s", p.Path, p.Version, err, string(out))
		}
	}
	return nil
}

// parseModuleReplaces returns the module-version `replace` directives in a go.mod
// keyed by the replaced (old) module path, mapping to the replacement path+version.
// Filesystem replacements (`=> ./local`) carry no version and are skipped — they
// are workspace-local modules, never part of the lock.
func parseModuleReplaces(goModPath string) (map[string]RequiredGoModule, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()

	replaces := make(map[string]RequiredGoModule)
	scanner := bufio.NewScanner(file)
	inBlock := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "replace (") {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}
		if !inBlock {
			if !strings.HasPrefix(line, "replace ") {
				continue
			}
			line = strings.TrimPrefix(line, "replace ")
		} else if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		parts := strings.SplitN(line, "=>", 2)
		if len(parts) != 2 {
			continue
		}
		oldFields := strings.Fields(parts[0])
		newFields := strings.Fields(parts[1])
		if len(oldFields) < 1 || len(newFields) < 2 {
			continue // filesystem replacement (no version) or malformed.
		}
		newPath := newFields[0]
		if strings.HasPrefix(newPath, "./") || strings.HasPrefix(newPath, "../") || strings.HasPrefix(newPath, "/") {
			continue // filesystem replacement.
		}
		replaces[oldFields[0]] = RequiredGoModule{Path: newPath, Version: newFields[1]}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read go.mod: %w", err)
	}
	return replaces, nil
}

// readGoSum parses a go.sum into a map keyed by "path version", capturing both the
// module zip hash and the go.mod hash for each. A missing file yields an empty map
// (a plugin with no external deps).
func readGoSum(goSumPath string) (map[string]goSumHashes, error) {
	file, err := os.Open(goSumPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]goSumHashes{}, nil
		}
		return nil, fmt.Errorf("failed to open go.sum: %w", err)
	}
	defer file.Close()

	out := make(map[string]goSumHashes)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Each line is: <path> <version>[/go.mod] h1:<hash>=
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 {
			continue
		}
		path, ver, hash := fields[0], fields[1], fields[2]
		isGoMod := strings.HasSuffix(ver, "/go.mod")
		ver = strings.TrimSuffix(ver, "/go.mod")
		key := path + " " + ver
		h := out[key]
		if isGoMod {
			h.goMod = hash
		} else {
			h.zip = hash
		}
		out[key] = h
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read go.sum: %w", err)
	}
	return out, nil
}
