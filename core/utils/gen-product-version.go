package tools

import (
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// productInfo mirrors core/utils/product.productInfo — the shape of
// core/product.json ({"version": "..."}). Duplicated here (rather than imported)
// to keep this dev build-tool independent of the runtime product package.
type productInfo struct {
	Version string `json:"version"`
}

// GenProductVersion writes core/product.json with the version copied from
// core/plugin.json (the core version). In release builds the software-release
// pipeline stamps product.json with the operator-set per-partner product version;
// in local dev that stamp never happens, so the reflex build calls this to drop a
// dev stand-in equal to the core version. The file is gitignored — it is a
// generated dev artifact, never committed. product.Version() then reads it like
// any other build, so dev and release share one code path (no env branching).
func GenProductVersion() error {
	core, err := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
	if err != nil {
		return err
	}
	out := filepath.Join(sdkutils.PathCoreDir, "product.json")
	return sdkutils.JsonWrite(out, productInfo{Version: core.Version})
}
