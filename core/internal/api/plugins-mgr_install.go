package api

import (
	"core/internal/modules/pluginreport"

	sdkplugin "sdk/api"
)

// progressEmitter reports an install's progress to the caller's handle. It never
// blocks and is safe to call from the install goroutine. The build/install
// helpers accept one (nil = no reporting, used by the meta/update paths that have
// no caller-facing channel).
type progressEmitter func(stage sdkplugin.PluginInstallStage, percent int, msg string)

// call invokes the emitter unless it is nil, so callers can emit unconditionally.
func (e progressEmitter) call(stage sdkplugin.PluginInstallStage, percent int, msg string) {
	if e != nil {
		e(stage, percent, msg)
	}
}

// pluginInstall is the IPluginInstall handle returned by PluginsMgr.InstallPlugin.
// The install runs in a goroutine that emits progress events and records the final
// error; the channel is closed and done is signalled exactly once, in finish.
type pluginInstall struct {
	pkg      string
	progress chan sdkplugin.PluginInstallProgress
	done     chan struct{}
	err      error
}

func newPluginInstall(pkg string) *pluginInstall {
	return &pluginInstall{
		pkg:      pkg,
		progress: make(chan sdkplugin.PluginInstallProgress, 32),
		done:     make(chan struct{}),
	}
}

func (p *pluginInstall) Progress() <-chan sdkplugin.PluginInstallProgress {
	return p.progress
}

func (p *pluginInstall) Done() error {
	<-p.done
	return p.err
}

// emit delivers an intermediate progress event without ever blocking the install
// goroutine: if the buffer is full (a slow or absent consumer) the event is
// dropped. Done() remains the authoritative result and the channel close signals
// completion, so dropping intermediate events is safe.
func (p *pluginInstall) emit(stage sdkplugin.PluginInstallStage, percent int, msg string) {
	select {
	case p.progress <- sdkplugin.PluginInstallProgress{Pkg: p.pkg, Stage: stage, Percent: percent, Message: msg}:
	default:
	}
}

// finish records the terminal result, emits the final event, and closes both
// channels. It is called exactly once by the install goroutine.
func (p *pluginInstall) finish(err error) {
	p.err = err
	ev := sdkplugin.PluginInstallProgress{Pkg: p.pkg, Stage: sdkplugin.PluginInstallStageDone, Percent: 100}
	if err != nil {
		ev.Stage = sdkplugin.PluginInstallStageFailed
		ev.Err = err
		ev.Message = err.Error()
	}
	// Best-effort like emit: a consumer that ranged to channel close then calls
	// Done() gets the result regardless of whether this terminal event is buffered.
	select {
	case p.progress <- ev:
	default:
	}
	close(p.progress)
	close(p.done)

	// On a successful install, nudge the cloud to re-read this machine's installed
	// plugins so the new (or updated) plugin shows up without waiting for the daily
	// report. Coalesced, so installing every member of a meta yields one report.
	if err == nil {
		pluginreport.ReportNowAsync()
	}
}
