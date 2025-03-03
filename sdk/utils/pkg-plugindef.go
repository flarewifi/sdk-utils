package sdkutils

const (
	PluginSrcGit    string = "git"
	PluginSrcStore  string = "store"
	PluginSrcSystem string = "system"
	PluginSrcLocal  string = "local"
)

type PluginMetadata struct {
	Package string
	Def     PluginSrcDef
}

type PluginSrcDef struct {
	Src                string // git | store | system | local
	StorePackage       string // if src is "store"
	StorePluginVersion string // if src is "store"
	StoreZipUrl        string // if src is "store"
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
	case PluginSrcSystem, PluginSrcLocal:
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
