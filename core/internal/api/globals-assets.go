package api

import (
	"fmt"
	"path/filepath"
)

type GlobalAssets struct {
	AdminJsHash   string
	AdminJsFiles  []string
	AdminCssHash  string
	AdminCssFiles []string

	PortalJsHash   string
	PortalJsFiles  []string
	PortalCssHash  string
	PortalCssFiles []string
}

type GlobalAssetsPaths struct {
	AdminJsSrc    string
	AdminCssHref  string
	PortalJsSrc   string
	PortalCssHref string
}

func GetAssetsPaths(manifest *GlobalAssets) GlobalAssetsPaths {
	return GlobalAssetsPaths{
		AdminJsSrc:    filepath.Join("/assets/globals", fmt.Sprintf("global-admin-%s.js", manifest.AdminJsHash)),
		AdminCssHref:  filepath.Join("/assets/globals", fmt.Sprintf("global-admin-%s.css", manifest.AdminCssHash)),
		PortalJsSrc:   filepath.Join("/assets/globals", fmt.Sprintf("global-portal-%s.js", manifest.PortalJsHash)),
		PortalCssHref: filepath.Join("/assets/globals", fmt.Sprintf("global-portal-%s.css", manifest.PortalCssHash)),
	}
}
