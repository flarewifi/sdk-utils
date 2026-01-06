package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func SyncGoVersion() {
	b, err := os.ReadFile(filepath.Join(sdkutils.PathAppDir, ".go-version"))
	if err != nil {
		panic(err)
	}

	goVersion := strings.TrimSpace(string(b))
	goVersion = strings.TrimPrefix(goVersion, "go")

	files := []string{
		"core/go.mod",
		"sdk/api/go.mod",
		"go.work.default",
	}

	for _, f := range files {
		file := filepath.Join(sdkutils.PathAppDir, f)
		if err := ReplaceGoVersion(goVersion, file); err != nil {
			panic(err)
		}
	}
}

// ReplaceGoVersion replaces the major and minor Go version strings in the file at the given path.
func ReplaceGoVersion(version string, path string) error {
	// Compile regular expressions for both version patterns
	re1 := regexp.MustCompile(`go1\.18(\.\d*)?`)
	re2 := regexp.MustCompile(`go 1\.18(\.\d*)?`)

	// Extract the major.minor and patch from the provided version
	versionRegex := regexp.MustCompile(`(\d+\.\d+)(\.\d+)?`)
	matches := versionRegex.FindStringSubmatch(version)
	if len(matches) == 0 {
		return fmt.Errorf("invalid version format: %s", version)
	}
	versionMajorMinor := matches[1] // e.g. "1.20"
	versionPatch := matches[2]      // e.g. ".3", or empty if no patch

	// Read the file content
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Replace only the major and minor versions, preserving any existing patch number
	newContent := re1.ReplaceAllStringFunc(string(content), func(match string) string {
		// Check if the matched string contains a patch number
		hasPatch := regexp.MustCompile(`\.\d+`).FindString(match) != ""
		if hasPatch {
			return versionMajorMinor + versionPatch // Append the patch if it exists in the replacement version
		}
		return versionMajorMinor // Do not append the patch if the original didn't have one
	})

	newContent = re2.ReplaceAllStringFunc(newContent, func(match string) string {
		// Check if the matched string contains a patch number
		hasPatch := regexp.MustCompile(`\.\d+`).FindString(match) != ""
		if hasPatch {
			return versionMajorMinor + versionPatch // Append the patch if it exists in the replacement version
		}
		return versionMajorMinor // Do not append the patch if the original didn't have one
	})

	// Write the updated content back to the file
	err = os.WriteFile(path, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
