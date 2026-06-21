/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import sdkutils "github.com/flarewifi/sdk-utils"

// PluginInstallStage names a phase of a plugin install. An install moves through
// these stages in order; only one terminal stage (Done or Failed) is reached.
type PluginInstallStage string

const (
	// PluginInstallStageResolving: looking up the store release / download URL.
	PluginInstallStageResolving PluginInstallStage = "resolving"
	// PluginInstallStageQueued: the server-side build is enqueued, waiting for a
	// binary bot to pick it up.
	PluginInstallStageQueued PluginInstallStage = "queued"
	// PluginInstallStageBuilding: the plugin .so is being compiled (in the cloud
	// for store plugins, on-device for git/local plugins).
	PluginInstallStageBuilding PluginInstallStage = "building"
	// PluginInstallStageDownloading: fetching the install-ready tarball.
	PluginInstallStageDownloading PluginInstallStage = "downloading"
	// PluginInstallStageInstalling: extracting, running migrations, copying into
	// the install path and registering the plugin live.
	PluginInstallStageInstalling PluginInstallStage = "installing"
	// PluginInstallStageDone: install completed successfully (terminal).
	PluginInstallStageDone PluginInstallStage = "done"
	// PluginInstallStageFailed: install failed; the event's Err is set (terminal).
	PluginInstallStageFailed PluginInstallStage = "failed"
)

// PluginInstallProgress is a single progress event emitted during an install.
type PluginInstallProgress struct {
	// Pkg is the package being installed.
	Pkg string
	// Stage is the current phase.
	Stage PluginInstallStage
	// Percent is a best-effort, monotonic 0..100 completion estimate. The cloud
	// build reports only coarse states, so percentages within the build phase are
	// approximate.
	Percent int
	// Message is an optional human-readable detail (raw, untranslated). Consumers
	// should switch on Stage for user-facing text and use Message for extra
	// context such as an error string.
	Message string
	// Err is set only on the terminal Failed event.
	Err error
}

// IPluginInstall is the handle returned by InstallPlugin. The install runs in the
// background as soon as InstallPlugin returns.
type IPluginInstall interface {
	// Progress streams install stage events. It is closed after the terminal
	// (Done/Failed) event, so a consumer can drain it with `for ev := range
	// h.Progress()`. Sends are best-effort: if the consumer is slow, intermediate
	// events may be dropped, but Done() always reports the authoritative result.
	Progress() <-chan PluginInstallProgress

	// Done blocks until the install finishes and returns the final error (nil on
	// success). It is safe to call without consuming Progress.
	Done() error
}

// IPluginsMgrApi is used to get data of installed plugins in the system.
type IPluginsMgrApi interface {

	// Find a plugin by name as defined in package.yml "name" field.
	FindByName(name string) (IPluginApi, bool)

	// Find a plugin by path as defined in package.yml "package" field.
	FindByPkg(pkg string) (IPluginApi, bool)

	// Returns all plugins installed in the system.
	Plugins() []IPluginApi

	// InstallPlugin installs a plugin from any source and registers it live
	// without requiring a server restart. The source is selected by def.Src:
	//   - store:            resolves the release download URL internally via the
	//     core RPC service (from def.StorePackage / def.StorePluginVersion) and
	//     installs from an encrypted scratch disk. The resolved URL is transient
	//     and is never persisted as part of the source definition.
	//   - git:              clones def.GitURL at def.GitRef (a branch, tag, or
	//     commit hash) into a temp directory.
	//   - local/system: installs from def.LocalPath, a source folder already on
	//     disk (it must contain a plugin.json); the source files are left in place.
	//
	// When def resolves to a store meta bundle, InstallPlugin expands it: every
	// member is installed (or adopted if already present) and a bundle record is
	// saved. UninstallPlugin reverses this (it detects and removes meta bundles too).
	//
	// The install runs in the background. InstallPlugin returns immediately with a
	// handle: range over handle.Progress() for stage events and call handle.Done()
	// for the final result. Callers that only want the result can ignore Progress
	// and call Done() directly (it blocks until the install finishes).
	InstallPlugin(def sdkutils.PluginSrcDef) IPluginInstall

	// UninstallPlugin removes a plugin or meta bundle through a single entry point.
	// A regular plugin is marked for removal on the next restart. A meta bundle is
	// detected automatically and its bundle record is removed, cascading to its
	// members: any member left owned by no remaining bundle and not installed
	// standalone is also marked for removal on the next restart.
	UninstallPlugin(pkg string) error

	// MetaPlugins returns all installed meta-plugin bundle records.
	MetaPlugins() ([]sdkutils.MetaPlugin, error)

	// MetaMembership reports which installed meta bundles own pkg and whether it
	// should be treated as a standalone install. A plugin installed on its own, or
	// owned by no meta, is standalone. When the plugins config cannot be read it
	// returns ([]string{}, true) — the safe default (no owners, treated standalone).
	MetaMembership(pkg string) (owners []string, standalone bool)

	// IsToBeRemoved returns true if the plugin has been marked for removal.
	IsToBeRemoved(pkg string) bool

	// HasPendingUpdate returns true if a downloaded update is waiting to be applied.
	HasPendingUpdate(pkg string) bool

	// SourceDef returns the source definition for an installed plugin —
	// where it came from and how it was installed (git / store / system /
	// local). Returns (zero-value, false) if the package is not installed.
	SourceDef(pkg string) (sdkutils.PluginSrcDef, bool)
}
