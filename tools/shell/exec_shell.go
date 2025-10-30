package shell

import (
	"errors"
	"io"
	"log"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type ExecOpts struct {
	User   *string
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

		if opts.User != nil {
			// Find the user to execute the command as
			targetUser, err := user.Lookup(*opts.User)
			if err != nil {
				return err
			}

			// Get the UID and GID for the target user
			uid, err := strconv.Atoi(targetUser.Uid)
			if err != nil {
				log.Fatalf("Failed to get UID for user '%s': %v", targetUser.Username, err)
			}
			gid, err := strconv.Atoi(targetUser.Gid)
			if err != nil {
				log.Fatalf("Failed to get GID for user '%s': %v", targetUser.Username, err)
			}

			// Set the process attributes to run as the target user
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Credential: &syscall.Credential{
					Uid: uint32(uid),
					Gid: uint32(gid),
				},
			}
		}
	}

	var stderr strings.Builder
	if !hasStderr {
		cmd.Stderr = &stderr
	}

	log.Printf("Executing '%s': %s\n", shell, command)
	if opts != nil && opts.Dir != "" {
		log.Printf("Executing in: %s\n", opts.Dir)
	}

	if err = cmd.Run(); err != nil {
		if !hasStderr && stderr.String() != "" {
			err = errors.New(stderr.String())
		}
	}

	return err
}
