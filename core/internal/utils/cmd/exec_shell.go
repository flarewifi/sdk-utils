package cmd

import (
	"errors"
	"io"
	"log"
	"os/exec"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type ExecOpts struct {
	Stdout io.Writer
	Stderr io.Writer
	Dir    string
	Env    []string
}

var shell string

func init() {
	var shells = []string{"/bin/ash", "/bin/bash", "/bin/zsh"}
	for _, s := range shells {
		if sdkutils.FsExists(s) {
			shell = s
			break
		}
	}
}

func execShell(command string, opts *ExecOpts) (err error) {
	hasStderr := false
	cmd := exec.Command(shell, "-c", command)

	if opts != nil {
		if opts.Stdout != nil {
			cmd.Stdout = opts.Stdout
		}
		if opts.Stderr != nil {
			hasStderr = true
			cmd.Stderr = opts.Stderr
		}
		if opts.Dir != "" {
			cmd.Dir = opts.Dir
		}
		if len(opts.Env) > 0 {
			cmd.Env = opts.Env
		}
	}

	var stderr strings.Builder
	if !hasStderr {
		cmd.Stderr = &stderr
	}

	log.Printf("Executing '%s': %s\n", shell, command)

	if err = cmd.Run(); err != nil {
		if !hasStderr && stderr.String() != "" {
			err = errors.New(stderr.String())
		}
	}

	return err
}
