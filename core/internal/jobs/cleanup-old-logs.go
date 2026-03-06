package jobs

import (
	"bufio"
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"core/db"
	"core/db/models"
	"core/utils/config"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	// Default retention period in days (used if config is not set)
	defaultLogRetentionDays = 3
	// Maximum lines to keep in flarehotspot.logs file
	maxLogFileLines = 500
	// Log file name
	logFileName = "flarehotspot.logs"
)

// StartLogCleanupScheduler starts a background goroutine that cleans up
// old logs based on configured retention period.
// In dev mode: runs every 5 seconds. In prod: runs every hour.
func StartLogCleanupScheduler(database *db.Database, mdls *models.Models) {
	go func() {
		if LogCleanupInterval < time.Hour {
			log.Printf("[LogCleanup] DEV MODE: Running every %v", LogCleanupInterval)
		} else {
			log.Printf("[LogCleanup] Scheduler started - will run every %v", LogCleanupInterval)
		}

		for {
			log.Printf("[LogCleanup] Next cleanup scheduled in %v", LogCleanupInterval)
			time.Sleep(LogCleanupInterval)

			// Read retention days from application config
			retentionDays := defaultLogRetentionDays
			appCfg, err := config.ReadApplicationConfig()
			if err == nil && appCfg.LogsRetentionDays > 0 {
				retentionDays = appCfg.LogsRetentionDays
			}

			performLogCleanup(database, mdls, retentionDays)
		}
	}()
}

// performLogCleanup executes the cleanup of old logs from database and truncates log file
func performLogCleanup(database *db.Database, mdls *models.Models, retentionDays int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("[LogCleanup] Starting cleanup of logs older than %d days", retentionDays)
	startTime := time.Now()

	// Get count before cleanup (for logging)
	countBefore, err := mdls.Log().CountOlderThan(ctx, retentionDays)
	if err != nil {
		log.Printf("[LogCleanup] ERROR: Failed to count old logs: %v", err)
		return
	}

	if countBefore == 0 {
		log.Printf("[LogCleanup] No logs older than %d days to clean up", retentionDays)
	} else {
		log.Printf("[LogCleanup] Found %d log(s) older than %d days", countBefore, retentionDays)

		// Perform cleanup
		err = mdls.Log().DeleteOlderThan(ctx, retentionDays)
		if err != nil {
			log.Printf("[LogCleanup] ERROR: Failed to delete old logs: %v", err)
			return
		}

		duration := time.Since(startTime)
		log.Printf("[LogCleanup] Successfully deleted %d old log(s) in %v",
			countBefore, duration.Round(time.Millisecond))

		// Get total remaining logs (for statistics)
		totalRemaining, err := mdls.Log().CountAll(ctx)
		if err == nil {
			log.Printf("[LogCleanup] Total remaining logs: %d", totalRemaining)
		}
	}

	// Truncate flarehotspot.logs file to max 500 lines
	truncateLogFile()
}

// truncateLogFile keeps only the last maxLogFileLines lines in the log file
func truncateLogFile() {
	logFilePath := filepath.Join(sdkutils.PathTmpDir, logFileName)

	if !sdkutils.FsExists(logFilePath) {
		return
	}

	// Read all lines from the file
	file, err := os.Open(logFilePath)
	if err != nil {
		log.Printf("[LogCleanup] ERROR: Failed to open log file for truncation: %v", err)
		return
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		log.Printf("[LogCleanup] ERROR: Failed to read log file: %v", err)
		return
	}

	// If file has fewer lines than max, no truncation needed
	if len(lines) <= maxLogFileLines {
		log.Printf("[LogCleanup] Log file has %d lines, no truncation needed", len(lines))
		return
	}

	// Keep only the last maxLogFileLines lines
	linesToRemove := len(lines) - maxLogFileLines
	lines = lines[linesToRemove:]

	// Write truncated content back to file
	file, err = os.Create(logFilePath)
	if err != nil {
		log.Printf("[LogCleanup] ERROR: Failed to create truncated log file: %v", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		writer.WriteString(line + "\n")
	}
	writer.Flush()

	log.Printf("[LogCleanup] Truncated log file from %d to %d lines", linesToRemove+maxLogFileLines, maxLogFileLines)
}

// RunLogCleanupNow executes cleanup immediately (useful for manual triggers or testing)
func RunLogCleanupNow(database *db.Database, mdls *models.Models, retentionDays int) {
	log.Printf("[LogCleanup] Manual cleanup triggered for logs older than %d days", retentionDays)
	performLogCleanup(database, mdls, retentionDays)
}
