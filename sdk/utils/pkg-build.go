package sdkutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GoBuildOpts struct {
	GoBinPath string
	WorkDir   string
	Env       []string
	// GoArch    string
	BuildTags string
	ExtraArgs []string
}

func BuildGoModule(gofile string, outfile string, opts GoBuildOpts) error {
	if opts.GoBinPath == "" {
		opts.GoBinPath = "go"
	}

	if opts.Env == nil {
		opts.Env = os.Environ()
	}

	fmt.Println("Building go module: " + StripRootPath(filepath.Join(opts.WorkDir, gofile)))

	goBin := opts.GoBinPath
	buildArgs := DefaultGoBuildArgs(opts.BuildTags)
	buildArgs = append(buildArgs, opts.ExtraArgs...)

	buildCmd := []string{"build"}
	buildCmd = append(buildCmd, buildArgs...)
	buildCmd = append(buildCmd, "-o", outfile, gofile)

	cmdstr := goBin
	for _, arg := range buildCmd {
		cmdstr += " " + arg
	}

	// if opts.GoArch != "" {
	// 	cmdstr = fmt.Sprintf("GOARCH=%s %s", opts.GoArch, cmdstr)
	// }

	fmt.Printf(`Build working directory: %s`+"\n", StripRootPath(opts.WorkDir))
	fmt.Printf("Executing: %s\n", cmdstr)

	var stderr strings.Builder
	cmd := exec.Command("sh", "-c", cmdstr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), opts.Env...)
	cmd.Dir = opts.WorkDir
	err := cmd.Run()

	if err != nil {
		return fmt.Errorf("Failed to build go module: %s\n%s", err, stderr.String())
	}

	fmt.Println("Module built successfully: " + StripRootPath(filepath.Join(opts.WorkDir, outfile)))
	return nil
}

// DefaultGoBuildArgs returns the go build arguments with tags: go build -tags=[tags]
func DefaultGoBuildArgs(tags string) []string {
	args := []string{}
	args = append(args, "-ldflags='-s -w'", "-trimpath", "-buildvcs=false")
	if tags != "" {
		args = append(args, fmt.Sprintf("-tags='%s'", RmRedundantWords(tags)))
	}

	return args
}
