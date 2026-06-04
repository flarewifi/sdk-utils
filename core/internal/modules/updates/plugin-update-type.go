package updates

// PluginUpdate describes a single installed plugin OR meta bundle on the Software
// Updates page. A bundle is represented by one PluginUpdate (IsMeta == true) keyed
// by the bundle package; its members are not listed individually.
//
// It is purely informational — plugins are refreshed by the system update, not
// individually. On the initial page render only CurrentVersion is set (see
// ListInstalledPlugins); after a "Check for updates" the cloud lookup also fills
// LatestVersion/HasUpdate (see CheckPluginUpdates).
//
// This type lives in a non-build-tagged file (not plugins.go, which is !mono) so
// the shared SoftwareUpdatesPage template can reference it in both mono and
// non-mono builds. The discovery logic that populates it stays in plugins.go.
type PluginUpdate struct {
	Package        string
	Name           string
	CurrentVersion string
	LatestVersion  string
	HasUpdate      bool
	// IsMeta marks this row as a meta bundle rather than a standalone plugin.
	IsMeta bool
}
