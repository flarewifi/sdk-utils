package jobs

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"time"

	"core/db"
	"core/db/models"
	"core/utils/config"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	defaultLogRetentionDays = 3
	maxLogFileLines         = 500
	logFileName             = "flarehotspot.log"
)

func StartLogCleanupScheduler(database *db.Database, mdls *models.Models) {
	go func() {
		for {
			time.Sleep(LogCleanupInterval)

			retentionDays := defaultLogRetentionDays
			appCfg, err := config.ReadApplicationConfig()
			if err == nil && appCfg.LogsRetentionDays > 0 {
				retentionDays = appCfg.LogsRetentionDays
			}

			performLogCleanup(database, mdls, retentionDays)
		}
	}()
}

func performLogCleanup(database *db.Database, mdls *models.Models, retentionDays int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	countBefore, err := mdls.Log().CountOlderThan(ctx, retentionDays)
	if err != nil {
		return
	}

	if countBefore != 0 {
		err = mdls.Log().DeleteOlderThan(ctx, retentionDays)
		if err != nil {
			return
		}
	}

	truncateLogFile()
}

func truncateLogFile() {
	logFilePath := filepath.Join(sdkutils.PathTmpDir, logFileName)

	if !sdkutils.FsExists(logFilePath) {
		return
	}

	file, err := os.Open(logFilePath)
	if err != nil {
		return
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		return
	}

	if len(lines) <= maxLogFileLines {
		return
	}

	linesToRemove := len(lines) - maxLogFileLines
	lines = lines[linesToRemove:]

	file, err = os.Create(logFilePath)
	if err != nil {
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		writer.WriteString(line + "\n")
	}
	writer.Flush()
}

func RunLogCleanupNow(database *db.Database, mdls *models.Models, retentionDays int) {
	performLogCleanup(database, mdls, retentionDays)
}
