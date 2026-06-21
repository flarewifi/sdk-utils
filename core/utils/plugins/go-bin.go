package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func GoBin() string {
	goCustomPath := os.Getenv("GO_CUSTOM_PATH")
	goCustomBin := filepath.Join(goCustomPath, "bin", "go")
	if sdkutils.FsExists(goCustomBin) {
		fmt.Println("Testing go binary: ", goCustomBin)
		testGo := exec.Command(goCustomBin, "env")
		err := testGo.Run()
		if err != nil {
			fmt.Println(err)
			fmt.Println("Error running custom go binary, fallback to system...")
			return "go"
		}
		fmt.Println("Using custom go binary: ", goCustomBin)
		return goCustomBin
	}

	fmt.Println("Using system go binary...")
	return "go"
}
