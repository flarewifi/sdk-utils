/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 :xa
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package sdkutils

import (
	"os"
	"path/filepath"
	"strings"
)

var (
	PathTmpDir           = getTmpDir()
	PathAppDir           = getRootDir()
	PathDataDir          = filepath.Join(PathAppDir, "data")
	PathCoreDir          = filepath.Join(PathAppDir, "core")
	PathDefaultsDir      = filepath.Join(PathAppDir, "defaults")
	PathConfigDir        = filepath.Join(PathDataDir, "config")
	PathLogsDir          = filepath.Join(PathAppDir, "logs")
	PathSdkDir           = filepath.Join(PathAppDir, "sdk")
	PathStorageDir       = filepath.Join(PathDataDir, "storage")
	PathPluginSystemDir  = filepath.Join(PathAppDir, "plugins", "system")
	PathPluginInstallDir = filepath.Join(PathAppDir, "plugins", "installed")
	PathPluginBackupsDir = filepath.Join(PathAppDir, "plugins", "backups")
	PathPluginUpdatesDir = filepath.Join(PathAppDir, "plugins", "updates")
	PathPluginLocalDir   = filepath.Join(PathDataDir, "plugins", "local")
	PathPluginCacheDir   = filepath.Join(PathConfigDir, "plugins", "cache")
	PathSystemUpdateDir  = filepath.Join(PathStorageDir, "system", "update")
	PathCacheDir         = filepath.Join(PathTmpDir, ".cache")
	PathIsUpdated        = filepath.Join(PathAppDir, ".updated")
	PathIsReverted       = filepath.Join(PathAppDir, ".reverted")
	PathServerUp         = "/tmp/.flare-up"
)

// StripRootPath removes the project root directory prefix from absolute paths
func StripRootPath(p string) string {
	return strings.Replace(p, PathAppDir+string(filepath.Separator), "", 1)
}

func getRootDir() string {
	if dir := os.Getenv("APP_DIR"); dir != "" {
		return dir
	}

	dir, err := os.Getwd()
	if err == nil && dir != "" {
		return dir
	}

	return "."
}

func getTmpDir() string {
	tmp := os.Getenv("APPTMP")
	if tmp == "" {
		return filepath.Join(getRootDir(), ".tmp")
	}
	return tmp
}
