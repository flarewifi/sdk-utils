package tools

import (
	"os"
	"os/exec"
	"path/filepath"

	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkpaths "github.com/flarehotspot/go-utils/paths"
)

const (
	SqlcVersion = "1.26.0"
)

func InstallSqlc() {
	if !sdkfs.Exists(sdkpaths.SqlcBin) {
		cmd := exec.Command("go", "build", "-buildvcs=false", "-o", sdkpaths.SqlcBin, filepath.Join(sdkpaths.SdkDir, "libs/sqlc-"+SqlcVersion+"/cmd/sqlc"))
		cmd.Dir = sdkpaths.AppDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			panic(err)
		}
	}
}
