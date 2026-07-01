// Package pkgmgr abstracts the on-device OpenWRT package manager so the rest of
// the core doesn't hard-code opkg. OpenWRT up to and including 24.10.x ships
// opkg; the newer line (>= 25.0.0 / snapshots) ships apk. The two managers take
// different verbs (`opkg install` vs `apk add`) and installed-query syntax, so
// the core selects one at runtime via Detect().
//
// All shell-outs go through core/utils/shell (the `cmd` alias) — never raw
// os/exec — so the `dev` build-tag stub keeps these as no-ops in the dev
// container where neither package manager exists.
package pkgmgr

import (
	"bytes"
	cmd "core/utils/shell"
	"os"
	"strings"
)

// Manager is a minimal package-manager interface covering the operations the
// core needs for plugin system_packages and bundled local packages.
type Manager interface {
	// Name returns the manager's binary name ("apk" or "opkg").
	Name() string
	// Update refreshes the package index.
	Update() error
	// Install installs the given packages from the configured feeds.
	Install(pkgs []string) error
	// InstallFiles installs local package files (.apk / .ipk) by path.
	InstallFiles(paths []string) error
	// IsInstalled reports whether an exact package name is installed.
	IsInstalled(pkg string) (bool, error)
}

// apkBinPaths are the locations the apk binary may live at on OpenWRT.
var apkBinPaths = []string{"/usr/bin/apk", "/sbin/apk", "/bin/apk", "/usr/sbin/apk"}

// Detect picks the package manager for the running machine. apk is used when its
// binary is present on disk (the newer OpenWRT line); otherwise opkg is assumed
// (the legacy default). Binary presence is more robust on-device than parsing
// the OpenWRT release version.
func Detect() Manager {
	for _, p := range apkBinPaths {
		if _, err := os.Stat(p); err == nil {
			return apkManager{}
		}
	}
	return opkgManager{}
}

// =============================================================================
// opkg (OpenWRT <= 24.10)
// =============================================================================

type opkgManager struct{}

func (opkgManager) Name() string { return "opkg" }

func (opkgManager) Update() error {
	return cmd.Exec("opkg update", &cmd.ExecOpts{Stdout: os.Stdout, Stderr: os.Stderr})
}

func (opkgManager) Install(pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}
	return cmd.Exec("opkg install "+strings.Join(pkgs, " "), &cmd.ExecOpts{Stdout: os.Stdout, Stderr: os.Stderr})
}

func (opkgManager) InstallFiles(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	return cmd.Exec("opkg install "+strings.Join(paths, " "), &cmd.ExecOpts{Stdout: os.Stdout, Stderr: os.Stderr})
}

func (opkgManager) IsInstalled(pkg string) (bool, error) {
	var output bytes.Buffer
	if err := cmd.ExecOutput("opkg list-installed", &output); err != nil {
		return false, err
	}
	// opkg list-installed lines look like "pkgname - version"; match the package
	// name exactly so e.g. "python3" doesn't satisfy "python3-light".
	for _, line := range strings.Split(output.String(), "\n") {
		name := strings.TrimSpace(line)
		if i := strings.IndexByte(name, ' '); i >= 0 {
			name = name[:i]
		}
		if name == pkg {
			return true, nil
		}
	}
	return false, nil
}

// =============================================================================
// apk (OpenWRT >= 25.0.0 / snapshots)
// =============================================================================

type apkManager struct{}

func (apkManager) Name() string { return "apk" }

func (apkManager) Update() error {
	return cmd.Exec("apk update", &cmd.ExecOpts{Stdout: os.Stdout, Stderr: os.Stderr})
}

func (apkManager) Install(pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}
	return cmd.Exec("apk add "+strings.Join(pkgs, " "), &cmd.ExecOpts{Stdout: os.Stdout, Stderr: os.Stderr})
}

func (apkManager) InstallFiles(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	// `apk add --allow-untrusted ./file.apk` installs a local package file.
	return cmd.Exec("apk add --allow-untrusted "+strings.Join(paths, " "), &cmd.ExecOpts{Stdout: os.Stdout, Stderr: os.Stderr})
}

func (apkManager) IsInstalled(pkg string) (bool, error) {
	// `apk info -e <pkg>` prints the package name if installed and nothing
	// otherwise, exiting non-zero when absent. The non-zero exit is the normal
	// "not installed" signal, so inspect the output rather than the error.
	var output bytes.Buffer
	_ = cmd.ExecOutput("apk info -e "+pkg, &output)
	for _, line := range strings.Split(output.String(), "\n") {
		if strings.TrimSpace(line) == pkg {
			return true, nil
		}
	}
	return false, nil
}
