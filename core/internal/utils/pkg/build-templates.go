package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-h/templ/generator"
	"github.com/a-h/templ/parser/v2"
	sdkfs "github.com/flarehotspot/go-utils/fs"
)

func BuildTemplates(pluginDir string) (err error) {
	var templateFiles []string
	if err = sdkfs.LsFiles(filepath.Join(pluginDir, "resources/views"), &templateFiles, true); err != nil {
		return
	}

	for _, file := range templateFiles {
		if filepath.Ext(file) == ".templ" {
			dir := filepath.Dir(file)
			filename := filepath.Base(file)
			out := filepath.Join(dir, strings.Replace(filename, ".templ", "_templ.go", 1))
			fmt.Println("Generating template:", file, "->", out)

			t, err := parser.Parse(file)
			if err != nil {
				fmt.Println("Error parsing template", file, err)
				return err
			}

			if sdkfs.Exists(out) {
				if err = os.Remove(out); err != nil {
					return err
				}
			}

			outfile, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Println("Error opening file:", err)
				return err
			}

			_, _, err = generator.Generate(t, outfile)
			if err != nil {
				outfile.Close() // Ensure the file is closed after writing
				fmt.Println("Error generating template", err)
				return err
			}

			outfile.Close() // Ensure the file is closed after writing
		}

		if strings.HasSuffix(file, "_templ.go") {
			defer removeDanglingTemplFile(file)
		}
	}

	return nil
}

func removeDanglingTemplFile(templgoFile string) (err error) {
	templFile := strings.Replace(templgoFile, "_templ.go", ".templ", 1)
	if !sdkfs.Exists(templFile) {
		err = os.Remove(templgoFile)
	}
	return
}
