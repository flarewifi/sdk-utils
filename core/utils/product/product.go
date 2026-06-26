// Package product exposes the machine's per-B2B-partner product version.
//
// The cloud software-release build stamps the operator-set product version into
// core/product.json (a file distinct from core/plugin.json). The machine reports
// THIS as its software-update version, so update-eligibility tracks the partner's
// own release lineage — independent of the core version (plugin.json "version"),
// which stays the ABI identity used for plugin .so compatibility.
//
// This is a leaf package (no core/internal imports) so both the machine API
// (IMachineApi.ProductVersion) and the updates module can read it without an
// import cycle.
package product

import (
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// productFile is the stamped file name in the core directory, beside plugin.json.
const productFile = "product.json"

type productInfo struct {
	Version string `json:"version"`
}

// Version returns the machine's product version. It prefers core/product.json
// (the stamped per-partner version) and falls back to the core/plugin.json
// version when product.json is absent or empty — older builds and dev checkouts
// that were never stamped, which then report their core version unchanged. Returns
// "" only if neither file is readable.
func Version() string {
	var info productInfo
	if err := sdkutils.JsonRead(filepath.Join(sdkutils.PathCoreDir, productFile), &info); err == nil && info.Version != "" {
		return info.Version
	}

	core, err := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
	if err != nil {
		return ""
	}
	return core.Version
}
