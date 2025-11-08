package plugins

import (
	"path/filepath"
	"strings"
	"tools/env"
	"tools/tags"

	"github.com/evanw/esbuild/pkg/api"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

func EsbuildJs(indexfile string, outfile string, target api.Target) (resulti api.BuildResult) {
	var sourcemap api.SourceMap
	sourcemap = api.SourceMapLinked
	if strings.Contains(outfile, "global") || tags.HasGoTag("mono") && env.GO_ENV == env.ENV_PRODUCTION {
		sourcemap = api.SourceMapNone
	}

	minify := env.GO_ENV == env.ENV_PRODUCTION

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{indexfile},
		Outfile:     outfile,
		Alias: map[string]string{
			"@flare/lib": filepath.Join(sdkutils.PathAppDir, "core/resources/assets/lib"),
		},
		Platform:          api.PlatformBrowser,
		Target:            target,
		EntryNames:        "[name]-[hash]",
		Sourcemap:         sourcemap,
		Bundle:            true,
		AllowOverwrite:    true,
		MinifyWhitespace:  minify,
		MinifyIdentifiers: minify,
		Write:             false,
	})

	return result
}

func EsbuildCss(indexfile string, outfile string) (result api.BuildResult) {
	var sourcemap api.SourceMap
	sourcemap = api.SourceMapLinked
	if strings.Contains(outfile, "global") || tags.HasGoTag("mono") && env.GO_ENV == env.ENV_PRODUCTION {
		sourcemap = api.SourceMapNone
	}

	minify := env.GO_ENV == env.ENV_PRODUCTION

	result = api.Build(api.BuildOptions{
		EntryPoints: []string{indexfile},
		Outfile:     outfile,
		Loader: map[string]api.Loader{
			".css":   api.LoaderCSS,
			".eot":   api.LoaderFile,
			".ttf":   api.LoaderFile,
			".otf":   api.LoaderFile,
			".woff":  api.LoaderFile,
			".woff2": api.LoaderFile,
			".svg":   api.LoaderFile,
			".jpg":   api.LoaderFile,
			".jpeg":  api.LoaderFile,
			".png":   api.LoaderFile,
			".gif":   api.LoaderFile,
		},
		EntryNames:        "[name]-[hash]",
		Sourcemap:         sourcemap,
		Bundle:            true,
		AllowOverwrite:    true,
		MinifyWhitespace:  minify,
		MinifyIdentifiers: minify,
		Write:             false,
	})

	return result
}
