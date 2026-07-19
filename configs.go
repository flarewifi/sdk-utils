package sdkutils

// AppConfig is the application configuration.
type AppConfig struct {
	// Examples: en, zh
	Lang string `json:"lang"`

	// Examples: USD, PH, CNY
	Currency string `json:"currency"`

	// Application secret key
	Secret string `json:"secret"`

	// Application channel: development, beta, stable
	Channel string `json:"channel"`

	// IANA timezone name (e.g. "Asia/Manila") the machine displays local times
	// in. Falls back to the server process's own local zone when unset or
	// invalid — see UtcToLocalTime.
	Timezone string `json:"timezone"`

	// Log retention period in days (default: 3)
	LogsRetentionDays int `json:"logs_retention_days"`

	// Enable logging to database (default: false)
	EnableLogging bool `json:"enable_logging"`

	// Maximum file size for plugin storage in bytes (default: 10MB)
	PluginMaxFileSize int64 `json:"plugin_max_file_size"`

	// Shared captive-portal hostname served locally with a valid, cloud-issued
	// certificate (split-horizon DNS, RFC 8910 advertisement, TLS SAN).
	//
	// NOTE: currently IGNORED
	CustomDomain string `json:"custom_domain"`
}

type DbConfig struct {
	SqlitePath string `json:"sqlite_path"`
}

const (
	PluginSrcGit    string = "git"
	PluginSrcStore  string = "store"
	PluginSrcSystem string = "system"
	PluginSrcLocal  string = "local"
)

type PluginsConfig struct {
	Metadata    []PluginMetadata `json:",omitempty"`
	MetaPlugins []MetaPlugin     `json:",omitempty"`
}

type PluginMetadata struct {
	Def     PluginSrcDef `json:",omitempty"`
	Package string       `json:",omitempty"`
	// Standalone is true when the user installed this package on its own
	// (via the normal install button), as opposed to it being pulled in only
	// as a member of a meta plugin. A standalone member survives meta uninstall.
	// Meta ownership itself is not stored here — it is derived from the meta
	// install records (MetaPlugin.Members), the single source of truth.
	Standalone bool `json:",omitempty"`
}

// MetaPlugin tracks an installed meta plugin. A meta has no plugin.so
// artifact of its own — it is a named bundle whose members are installed
// individually. The record lets us list the meta as a single entry and cascade
// its uninstall to members it owns.
type MetaPlugin struct {
	Package string   `json:",omitempty"`
	Name    string   `json:",omitempty"`
	Version string   `json:",omitempty"`
	Members []string `json:",omitempty"`
}

type PluginSrcDef struct {
	Src                string `json:",omitempty"` // git | store | system | local
	StorePackage       string `json:",omitempty"` // if src is "store"
	StorePluginVersion string `json:",omitempty"` // if src is "store"
	GitURL             string `json:",omitempty"` // if src is "git"
	GitRef             string `json:",omitempty"` // can be a branch, tag or commit hash
	LocalPath          string `json:",omitempty"` // if src is "local or system"
}

func (def PluginSrcDef) String() string {
	switch def.Src {
	case PluginSrcGit:
		return def.GitURL
	case PluginSrcStore:
		return def.StorePackage + "@" + def.StorePluginVersion
	case PluginSrcLocal:
		return def.LocalPath
	default:
		return "unknown plugin source: " + def.Src
	}
}

func (def PluginSrcDef) Equal(compare PluginSrcDef) bool {
	if (def.Src == PluginSrcLocal || def.Src == PluginSrcSystem) &&
		compare.Src == def.Src &&
		StripRootPath(def.LocalPath) == StripRootPath(compare.LocalPath) {
		return true
	}
	if def.Src == PluginSrcGit && compare.Src == PluginSrcGit && NeutralizeGitURL(def.GitURL) == NeutralizeGitURL(compare.GitURL) {
		return true
	}
	if def.Src == PluginSrcStore && compare.Src == PluginSrcStore && def.StorePackage == compare.StorePackage {
		return true
	}
	return false
}
