/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"errors"
	"net/http"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// ErrPaymentRequired is returned by InstallPlugin when a paid plugin is being
// installed on a machine that is not purchased to it. Callers can detect it with
// errors.Is to show a "purchase required" prompt instead of a generic failure.
var ErrPaymentRequired = errors.New("payment required")

// StoreReleaseError carries an operator-safe reason from the cloud store for why a
// plugin release could not be resolved for install — e.g. the requested version is
// published on a different release channel than the machine's, or it does not exist
// at all. Reason is curated server-side and safe to render in the admin UI: unlike a
// raw RPC/transport error it never contains the cloud endpoint URL, domains, or
// secrets. Callers detect it with errors.As to show the real cause instead of a
// generic "installation failed" message.
type StoreReleaseError struct {
	Reason string
}

func (e *StoreReleaseError) Error() string { return e.Reason }

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

	// RequiresReboot reports whether this install/update only takes effect
	// after the machine reboots. Meaningful only once Done() has returned nil:
	// false for a fresh install (registered live, no reboot needed) or a
	// meta-bundle update (which reboots itself internally when needed); true
	// when this call updated an ALREADY-LOADED standalone store plugin, whose
	// overwritten .so was staged to disk but cannot be hot-reloaded into the
	// running process (Go's plugin.Open cannot reload a .so already mapped
	// into this process). Always false while the install is still in progress
	// or if it failed.
	RequiresReboot() bool
}

// PluginPurchaseInfo describes whether this machine may install a plugin and
// at what price, as resolved by the cloud store. Used to gate installs and to
// drive the store marketplace UI ("Free" / price / "Purchase required").
type PluginPurchaseInfo struct {
	Package              string
	Purchased            bool // free OR already paid/covered for this machine
	IsFree               bool
	PricingType          string // "one_time" | "subscription"
	SubscriptionInterval string // "monthly" | "yearly" (when subscription)
	PriceUsdCents        int64
	LocalCurrency        string // developer-chosen currency, if any
	LocalPriceCents      int64
	// Price resolved into the machine owner's own currency (buyer's country ->
	// local price if it matches LocalCurrency, else USD). This is the amount the
	// checkout will actually charge, so render this for the buyer rather than
	// re-deriving from PriceUsdCents/LocalPriceCents. Empty/0 from an older cloud.
	DisplayCurrency   string
	DisplayPriceCents int64
	ExpiresAt         int64  // unix seconds; 0 = none / perpetual
	Available         bool   // false if some issue prevents install (e.g. the developer disabled the plugin). See Reason.
	Reason            string // human-readable explanation when not available or not purchased
}

// RequiresPayment reports whether install should be blocked pending purchase: a
// paid plugin this machine is not purchased to. Availability is a SEPARATE concern
// — an unavailable plugin (Available == false) cannot be installed at any price, so
// check Available first (install paths gate it ahead of this).
func (e PluginPurchaseInfo) RequiresPayment() bool {
	return !e.IsFree && !e.Purchased
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
	// For a store plugin, InstallPlugin validates payment up front: if the plugin
	// is paid and this machine is not purchased to it, InstallPlugin returns
	// (nil, ErrPaymentRequired) without starting an install. Use errors.Is(err,
	// ErrPaymentRequired) to redirect the owner to checkout. (The cloud also
	// withholds the download as an independent backstop.)
	//
	// Otherwise the install runs in the background and InstallPlugin returns a
	// handle with a nil error: range over handle.Progress() for stage events and
	// call handle.Done() for the final result. Callers that only want the result
	// can ignore Progress and call Done() directly (it blocks until done).
	InstallPlugin(def sdkutils.PluginSrcDef) (IPluginInstall, error)

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

	// CheckPurchase asks the cloud store whether this machine may install the
	// given plugin and at what price. Returns the resolved pricing/purchase so
	// callers can gate installs and render store UI. The install path also enforces
	// this independently (the cloud withholds the download for unpurchased paid
	// plugins), so this is for UX, not the only gate.
	CheckPurchase(pkg string) (PluginPurchaseInfo, error)

	// GetPurchaseURL builds the cloud checkout URL for purchasing pkg. After the
	// machine owner pays, the cloud redirects the browser back to callbackRouteName
	// — a route registered by the CALLING plugin on this machine — where the plugin
	// should call InstallPlugin(pkg) to complete the purchase.
	//
	// pairs are forwarded to the callback route exactly like UrlForRoute (key, value,
	// key, value, ...) to fill its path parameters, e.g.
	//   GetPurchaseURL(r, "com.flarego.cloud-sync", "admin:store:install:pkg", "pkg", "com.flarego.cloud-sync")
	// The resolved callback URL also always carries a "?pkg=<pkg>" query param so the
	// handler knows what to install regardless of the route's own parameters.
	//
	// r supplies the machine's browser-facing scheme://host so the cloud redirect
	// lands back on this device (the same host the owner used to reach the store),
	// not on loopback. The machine id and cloud checkout host are resolved
	// internally. Returns an error if pkg/callbackRouteName are empty or the
	// callback route is not registered.
	GetPurchaseURL(r *http.Request, pkg string, callbackRouteName string, pairs ...string) (string, error)
}
