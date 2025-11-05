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
}

type DbConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Database   string `json:"database"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	SslMode    string `json:"sslmode"`
	SqlitePath string `json:"sqlite_path"`
}

const (
	PluginSrcGit    string = "git"
	PluginSrcStore  string = "store"
	PluginSrcSystem string = "system"
	PluginSrcLocal  string = "local"
	PluginSrcZip    string = "zip"
)

type PluginsConfig struct {
	Recompile []string
	Metadata  []PluginMetadata
}

type PluginMetadata struct {
	Def     PluginSrcDef
	Package string
}

type PluginSrcDef struct {
	Src                string // git | store | system | local
	StorePackage       string // if src is "store"
	StorePluginVersion string // if src is "store"
	StoreZipURL        string // if src is "store"
	GitURL             string // if src is "git"
	GitRef             string // can be a branch, tag or commit hash
	LocalPath          string // if src is "local or system"
}

func (def PluginSrcDef) String() string {
	switch def.Src {
	case PluginSrcGit:
		return def.GitURL
	case PluginSrcStore:
		return def.StorePackage + "@" + def.StorePluginVersion
	case PluginSrcSystem, PluginSrcLocal, PluginSrcZip:
		return def.LocalPath
	default:
		return "unknown plugin source: " + def.Src
	}
}

func (def PluginSrcDef) Equal(compare PluginSrcDef) bool {
	if (def.Src == PluginSrcLocal || def.Src == PluginSrcSystem || def.Src == PluginSrcZip) &&
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
