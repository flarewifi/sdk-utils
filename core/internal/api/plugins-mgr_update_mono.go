//go:build mono

// In a mono build the core and plugins are compiled together server-side and
// updated as one system release, so there is no per-plugin prebuild to fetch:
// the cloud refuses RequestPluginBuild for mono machines and a plugin .so could
// not be loaded into the statically-linked mono binary anyway. This twin exists
// so the shared store-install path in plugins-mgr.go compiles under mono and
// fails with an honest error instead of a missing-toolchain crash.
package api

import "errors"

func (self *PluginsMgr) fetchPrebuiltPluginURL(pkg, version, coreVersion string, emit progressEmitter) (string, error) {
	return "", errors.New("store plugin installs are not supported on monolithic builds; plugins ship with the system release")
}
