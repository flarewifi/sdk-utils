package pkg

import (
	"core/internal/utils/git"
)

type PluginSrcDef struct {
	Src                string // git | store | system | local | zip
	StorePackage       string // if src is "store"
	StorePluginVersion string // if src is "store"
	StoreZipUrl        string // if src is "store"
	GitURL             string // if src is "git"
	GitRef             string // can be a branch, tag or commit hash
	LocalZipFile       string // if src is "zip"
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
	case PluginSrcZip:
		return def.LocalPath
	default:
		return "unknown"
	}
}

func (def PluginSrcDef) Equal(compare PluginSrcDef) bool {
	if (def.Src == PluginSrcLocal || def.Src == PluginSrcSystem) && def.LocalPath == compare.LocalPath {
		return true
	}
	if def.Src == PluginSrcGit && git.NeutralizeUrl(def.GitURL) == git.NeutralizeUrl(compare.GitURL) {
		return true
	}
	return false
}
