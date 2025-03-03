package tools

import (
	"errors"
	"fmt"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type BuildOutput struct {
	OutputDirName string
	Files         []string
	CustomFiles   []CustomFiles
}

type CustomFiles struct {
	Src  string
	Dest string
}

type metajson struct {
	GoVersion string   `json:"go_version"`
	GoArch    string   `json:"go_arch"`
	OutputDir string   `json:"output_dir"`
	OutputZip string   `json:"output_zip"`
	Files     []string `json:"files"`
}

func (b *BuildOutput) Run() error {
	if err := sdkutils.FsEmptyDir(b.outputPath()); err != nil {
		return err
	}

	contentList := []string{}
	for _, entry := range b.Files {
		srcPath := filepath.Join(sdkutils.PathAppDir, entry)
		destPath := filepath.Join(b.outputPath(), entry)
		if err := b.copy(srcPath, destPath); err != nil {
			panic(err)
		}
		contentList = append(contentList, entry)
	}

	for _, entry := range b.CustomFiles {
		srcPath := filepath.Join(sdkutils.PathAppDir, entry.Src)
		destPath := filepath.Join(b.outputPath(), entry.Dest)
		if err := b.copy(srcPath, destPath); err != nil {
			panic(err)
		}
		contentList = append(contentList, entry.Dest)
	}

	// new implementation using tar.gz
	if err := sdkutils.CompressTar(b.outputPath(), b.targzFilePath()); err != nil {
		return err
	}

	md := metajson{
		GoVersion: sdkutils.GO_VERSION,
		GoArch:    sdkutils.GOARCH,
		OutputDir: b.outputPath(),
		OutputZip: b.targzFilePath(),
		Files:     contentList,
	}

	if err := sdkutils.JsonWrite(b.metadataPath(), md); err != nil {
		return err
	}

	return nil
}

func (b *BuildOutput) copy(srcPath string, destPath string) error {
	fmt.Printf("Copying '%s' -> '%s'\n", srcPath, destPath)

	if !sdkutils.FsExists(srcPath) {
		return errors.New("File does not exist: " + srcPath)
	}

	if sdkutils.FsIsFile(srcPath) {
		if err := sdkutils.FsCopyFile(srcPath, destPath); err != nil {
			return err
		}
	} else if sdkutils.FsIsDir(srcPath) {
		if err := sdkutils.FsCopyDir(srcPath, destPath, nil); err != nil {
			return err
		}
	} else {
		return errors.New("Unknown file type: " + srcPath)
	}
	return nil
}

func (b *BuildOutput) outputPath() string {
	return filepath.Join(sdkutils.PathAppDir, "output", b.OutputDirName)
}

func (b *BuildOutput) targzFilePath() string {
	return filepath.Join(b.outputPath() + ".tar.gz")
}

func (b *BuildOutput) metadataPath() string {
	return filepath.Join(sdkutils.PathAppDir, "output/metadata.json")
}
