package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func MakePluginMainMono(pluginDir string) {
	mainFile := filepath.Join(pluginDir, "main.go")
	mainData, err := os.ReadFile(mainFile)
	if err != nil {
		panic(err)
	}
	newMainContent := addBuildTags(string(mainData), "!mono")

	if string(mainData) != newMainContent {
		err = os.WriteFile(mainFile, []byte(newMainContent), 0644)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s has been updated.\n", mainFile)
	}

	// Create mono version of main.go
	createMonoFile(pluginDir)
}

func addBuildTags(mainContent string, tag string) string {
	buildTagReg := regexp.MustCompile(`\s*\/\/\s*go:build\s+(.+)`)
	tagMatches := buildTagReg.FindAllStringSubmatch(mainContent, -1)
	var existingTags string
	if len(tagMatches) > 0 {
		existingTags = tagMatches[0][1]
	}

	hasBuildTag := buildTagReg.MatchString(mainContent)
	alreadyTagged := strings.Contains(existingTags, tag)

	newMainContent := mainContent
	if !hasBuildTag {
		newMainContent = fmt.Sprintf("//go:build %s\n\n%s", tag, mainContent)
	} else {
		if !alreadyTagged {
			newBuildTags := existingTags + " " + tag
			newMainContent = strings.ReplaceAll(mainContent, existingTags, newBuildTags)
		}
	}

	return newMainContent
}

func createMonoFile(pluginDir string) string {
	mainFile := filepath.Join(pluginDir, "main.go")
	mainData, err := os.ReadFile(mainFile)
	if err != nil {
		panic(err)
	}
	mainContent := string(mainData)
	packageReg := regexp.MustCompile(`package\s+(\w+)`)
	monoPackageName := sdkutils.Slugify(filepath.Base(pluginDir), "_")
	newMainContent := packageReg.ReplaceAllString(mainContent, fmt.Sprintf("package %s", monoPackageName))
	newMainContent = fmt.Sprintf("%s\n%s", AUTO_GENERATED_HEADER, newMainContent)
	newMainContent = strings.ReplaceAll(newMainContent, "!mono", "mono")
	newMainContent = addBuildTags(newMainContent, "mono")

	// remove main func
	mainFuncReg := regexp.MustCompile(`(g?)func\s+main\s*\(\s*\)\s*\{\s*\}`)
	newMainContent = mainFuncReg.ReplaceAllString(newMainContent, "")

	monoFile := filepath.Join(pluginDir, "main_mono.go")

	var monoContent string
	if b, err := os.ReadFile(monoFile); err == nil {
		monoContent = string(b)
	}

	if newMainContent != monoContent {
		err = os.WriteFile(monoFile, []byte(newMainContent), 0644)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s has been created.\n", monoFile)
	}

	return newMainContent
}
