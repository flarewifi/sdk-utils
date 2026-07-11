package tools

import (
	"encoding/json"
	"path/filepath"

	"core/utils/crypt"
	"core/utils/env"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// productInfo mirrors the shape of core/utils/product.productInfo — the shape of
// core/product.json. Duplicated here (rather than imported) to keep this dev
// build-tool independent of the runtime product package.
type productInfo struct {
	Version string `json:"version"`
	Data    string `json:"data"`
}

// devEncryptedFields mirrors core/utils/product.encryptedFields, the plaintext
// shape encrypted into productInfo.Data. The cloud's MachineActivation/
// UpdateMachineInfo require brand_id/device_model/device_config to be non-empty
// (see flarewifi_v3_srv.go's validateMachineInfoV3), so local dev needs a stand-in
// here too — not just a dev version string. Values match the seeded local dev
// "Flarewifi" B2bPartner (db/seeds.rb's brand_id) and a real device slug from
// go/builder/imagebuilder/devices/arm64-generic; release builds stamp the
// operator's real values instead (see go/builder's writeProductVersion).
type devEncryptedFields struct {
	BrandId      string `json:"brand_id"`
	DeviceModel  string `json:"device_model"`
	DeviceConfig string `json:"device_config"`
}

const (
	devBrandId      = "269e515a-b91f-4bf0-a457-5db04b216751"
	devDeviceModel  = "arm64-generic"
	devDeviceConfig = "25.12.5-wan-lan-mono.yml"
)

// GenProductVersion writes core/product.json with the version copied from
// core/plugin.json (the core version), plus an encrypted Data blob carrying the dev
// stand-in fields above. In release builds the software-release pipeline stamps
// product.json with the operator-set per-partner version and real fields; in local
// dev that stamp never happens, so the reflex build calls this to drop dev
// equivalents. The file is gitignored — it is a generated dev artifact, never
// committed. product.Version()/.BrandId()/etc. then read it like any other build,
// so dev and release share one code path (no env branching). Encryption uses
// env.RPC_TOKEN from the SAME build (dev/devkit/staging build tag), matching
// whatever key the running core binary will decrypt with.
func GenProductVersion() error {
	core, err := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
	if err != nil {
		return err
	}

	fields, err := json.Marshal(devEncryptedFields{
		BrandId:      devBrandId,
		DeviceModel:  devDeviceModel,
		DeviceConfig: devDeviceConfig,
	})
	if err != nil {
		return err
	}

	encrypted, err := crypt.EncryptToken(string(fields), env.RPC_TOKEN)
	if err != nil {
		return err
	}

	out := filepath.Join(sdkutils.PathCoreDir, "product.json")
	return sdkutils.JsonWrite(out, productInfo{Version: core.Version, Data: encrypted})
}
