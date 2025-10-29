//go:build !dev

package shell

import (
	"io"
)

func Exec(command string, opts *ExecOpts) error {
	return execShell(command, opts)
}

func ExecOutput(command string, out io.Writer) (err error) {
	return execShell(command, &ExecOpts{Stdout: out})
}
