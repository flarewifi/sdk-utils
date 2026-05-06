package boot

import (
	"log"
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const logsFile = "flarehotspot.log"

func CleanUpLogs() {
	log.Println("Cleaning up logs...")
	logsPath := filepath.Join(sdkutils.PathTmpDir, logsFile)
	if sdkutils.FsExists(logsPath) {
		os.WriteFile(logsPath, []byte{}, 0644)
	}
}
