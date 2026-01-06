package shell

import "io"

type ExecOpts struct {
	User   *string
	Stdout io.Writer
	Stderr io.Writer
	Dir    string
	Env    []string
}
