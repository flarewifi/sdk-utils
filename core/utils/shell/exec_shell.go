//go:build !dev

package shell

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

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
				return err
			}
			gid, err := strconv.Atoi(targetUser.Gid)
			if err != nil {
				return err
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

	if err = cmd.Run(); err != nil {
		if !hasStderr && stderr.String() != "" {
			err = errors.New(stderr.String())
		}
	}

	return err
}

// ExecWithContext executes shell command with context cancellation support
func ExecWithContext(ctx context.Context, command string, opts *ExecOpts) error {
	hasStderr := false
	cmd := exec.CommandContext(ctx, shell, "-c", command)

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
				return err
			}
			gid, err := strconv.Atoi(targetUser.Gid)
			if err != nil {
				return err
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

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("command timed out: %w", err)
		}
		if !hasStderr && stderr.String() != "" {
			err = errors.New(stderr.String())
		}
		return err
	}

	return nil
}
