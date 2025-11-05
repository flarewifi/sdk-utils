package sdkutils

import (
	"errors"
	"fmt"
	"path/filepath"
)

type BuildOutput struct {
	SourceDir string
	OutputDir string
	Files     []string
	Custom    []BuildOutputCustomEntry
}

type BuildOutputCustomEntry struct {
	Src  string
	Dest string
}

type BuildOutputMeta struct {
	GoVersion string   `json:"go_version"`
	GoArch    string   `json:"go_arch"`
	OutputDir string   `json:"output_dir"`
	OutputZip string   `json:"output_zip"`
	Files     []string `json:"files"`
}

func ReadBuildOutput(outdir string) (meta BuildOutputMeta, err error) {
	if !FsExists(outdir) {
		return meta, errors.New("Output directory does not exist: " + outdir)
	}

	err = JsonRead(filepath.Join(outdir, "metadata.json"), &meta)
	return
}

func (b *BuildOutput) Run() error {
	srcDir := PathAppDir
	if b.SourceDir != "" {
		srcDir = b.SourceDir
	}

	if err := FsEmptyDir(b.OutputDir); err != nil {
		return err
	}

	contentList := []string{}
	for _, entry := range b.Files {
		srcPath := filepath.Join(srcDir, entry)
		destPath := filepath.Join(b.OutputDir, entry)
		if err := b.copy(srcPath, destPath); err != nil {
			panic(err)
		}
		contentList = append(contentList, entry)
	}

	for _, entry := range b.Custom {
		srcPath := filepath.Join(PathAppDir, entry.Src)
		destPath := filepath.Join(b.OutputDir, entry.Dest)
		if err := b.copy(srcPath, destPath); err != nil {
			panic(err)
		}
		contentList = append(contentList, entry.Dest)
	}

	// new implementation using tar.gz
	if err := CompressTar(b.OutputDir, b.targzFilePath()); err != nil {
		return err
	}

	md := BuildOutputMeta{
		GoVersion: GO_VERSION,
		GoArch:    GOARCH,
		OutputDir: b.OutputDir,
		OutputZip: b.targzFilePath(),
		Files:     contentList,
	}

	if err := JsonWrite(b.metadataPath(), md); err != nil {
		return err
	}

	return nil
}

func (b *BuildOutput) copy(srcPath string, destPath string) error {
	fmt.Printf("Copying '%s' -> '%s'\n", srcPath, destPath)

	if !FsExists(srcPath) {
		return errors.New("File does not exist: " + srcPath)
	}

	return FsCopy(srcPath, destPath)
}

func (b *BuildOutput) targzFilePath() string {
	basename := filepath.Base(b.OutputDir)
	basedir := filepath.Dir(b.OutputDir)
	return filepath.Join(basedir, basename+".tar.gz")
}

func (b *BuildOutput) metadataPath() string {
	return filepath.Join(b.OutputDir, "metadata.json")
}
