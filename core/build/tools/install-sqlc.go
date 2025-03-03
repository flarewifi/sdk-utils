package tools

import (
	"os"
	"os/exec"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	SqlcVersion = "1.26.0"
)

func InstallSqlc() {
	if !sdkutils.FsExists(sdkutils.PathSqlcBin) {
		cmd := exec.Command("go", "build", "-buildvcs=false", "-o", sdkutils.PathSqlcBin, filepath.Join(sdkutils.PathSdkDir, "libs/sqlc-"+SqlcVersion+"/cmd/sqlc"))
		cmd.Dir = sdkutils.PathAppDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			panic(err)
		}
	}
}
