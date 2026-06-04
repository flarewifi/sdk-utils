//go:build mono

package tools

// StartScriptSrc returns the source filename of the boot script to ship as the
// device's start.sh. Mono builds use the wipe-and-restore start-mono.sh (the whole
// app is replaced from a single release tarball).
func StartScriptSrc() string { return "start-mono.sh" }
