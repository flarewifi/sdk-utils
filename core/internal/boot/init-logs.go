package boot

import (
	"log"
	"os"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const logsPath = "/opt/flarehotspot/tmp/flarehotspot.logs"

func CleanUpLogs() {
	log.Println("Cleaning up logs...")
	if sdkutils.FsExists(logsPath) {
		os.WriteFile(logsPath, []byte{}, 0644)
	}
}
