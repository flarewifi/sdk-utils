//go:build !mono

package tools

// StartScriptSrc returns the source filename of the boot script to ship as the
// device's start.sh. Non-mono builds use the staged-overlay applier (the default
// start.sh), which applies per-package updates from data/storage/system/updates/{pkg}
// on boot. Mono builds use the wipe-and-restore start-mono.sh (see
// start-script_mono.go).
func StartScriptSrc() string { return "start.sh" }
