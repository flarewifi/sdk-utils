// Package product exposes the machine's per-B2B-partner product version and
// its brand_id/device_config update-identity fields.
//
// The cloud software-release build stamps the operator-set product version into
// core/product.json (a file distinct from core/plugin.json). The machine reports
// THIS as its software-update version, so update-eligibility tracks the partner's
// own release lineage — independent of the core version (plugin.json "version"),
// which stays the ABI identity used for plugin .so compatibility.
//
// brand_id/device_config are stamped into the SAME file, AES-GCM encrypted
// (keyed on the shared RPC_TOKEN secret) — see go/builder's writeProductVersion.
// They're restamped on every release build, so (unlike os_release.json) they
// track the machine's CURRENTLY installed release, which real update-eligibility/
// product-transfer matching needs. device_model stays in os_release.json instead
// (see that package) since it must remain stable for the device's physical
// lifetime, not track its currently installed release.
//
// This is a leaf package (no core/internal imports) so both the machine API
// (IMachineApi.ProductVersion/.DeviceModel) and the updates module can read it
// without an import cycle. core/utils/env and core/utils/crypt are sibling leaf
// packages, so importing them here doesn't introduce one either.
package product

import (
	"encoding/json"
	"path/filepath"
	"sync"

	"core/utils/crypt"
	"core/utils/env"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// productFile is the stamped file name in the core directory, beside plugin.json.
const productFile = "product.json"

// readInfo/decryptedFields are cached process-lifetime: core/product.json is
// stamped once per build and never rewritten at runtime (an OTA update replaces
// it on disk but only takes effect after a reboot, which restarts the process),
// so re-reading the file and re-running AES-GCM decrypt on every call would be
// wasted CPU for a value that can't change.
var (
	infoOnce   sync.Once
	cachedInfo sdkutils.ProductInfo
	cachedOK   bool

	fieldsOnce   sync.Once
	cachedFields sdkutils.ProductFields
)

// Version returns the machine's product version. It prefers core/product.json
// (the stamped per-partner version) and falls back to the core/plugin.json
// version when product.json is absent or empty — older builds and dev checkouts
// that were never stamped, which then report their core version unchanged. Returns
// "" only if neither file is readable.
func Version() string {
	info, ok := readInfo()
	if ok && info.Version != "" {
		return info.Version
	}

	core, err := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
	if err != nil {
		return ""
	}
	return core.Version
}

// BrandId returns the machine's currently-installed release's brand_id, decrypted
// from core/product.json. Returns "" on any read/decrypt error or an empty/absent
// Data field — no fallback to os_release.json (it no longer carries this field).
func BrandId() string {
	return decryptedFields().BrandId
}

// DeviceConfig returns the machine's currently-installed release's device_config,
// decrypted from core/product.json. See BrandId for the no-fallback rationale.
func DeviceConfig() string {
	return decryptedFields().DeviceConfig
}

func readInfo() (sdkutils.ProductInfo, bool) {
	infoOnce.Do(func() {
		var info sdkutils.ProductInfo
		if err := sdkutils.JsonRead(filepath.Join(sdkutils.PathCoreDir, productFile), &info); err == nil {
			cachedInfo = info
			cachedOK = true
		}
	})
	return cachedInfo, cachedOK
}

func decryptedFields() sdkutils.ProductFields {
	fieldsOnce.Do(func() {
		info, ok := readInfo()
		if !ok || info.Data == "" {
			return
		}

		plaintext, err := crypt.DecryptToken(info.Data, env.RPC_TOKEN)
		if err != nil {
			return
		}

		var fields sdkutils.ProductFields
		if err := json.Unmarshal([]byte(plaintext), &fields); err != nil {
			return
		}
		cachedFields = fields
	})
	return cachedFields
}
