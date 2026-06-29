// Command minify-translations rewrites every resources/translations/<lang>.json
// under a target directory into the compact single-line form. It is the
// production build step that replaced the legacy per-language .tar.gz compression
// (see core/utils/translations.MinifyAllCatalogs): per-language JSON is tiny, so
// minification is all the size reduction the device needs.
//
// With no argument it minifies the whole app dir (sdkutils.PathAppDir). The
// non-mono software-release build runs it from the build dir with no argument so
// a single pass covers core/resources, plugins/installed and the bundled local
// plugin sources — every catalog that ships. An explicit directory argument
// narrows it to one staged tree (e.g. a single plugin's output-stage dir during a
// standalone plugin build).
package main

import (
	"core/utils/translations"
	"fmt"
	"os"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func main() {
	root := sdkutils.PathAppDir
	if len(os.Args) > 1 && os.Args[1] != "" {
		root = os.Args[1]
	}
	if err := translations.MinifyAllCatalogs(root); err != nil {
		panic(fmt.Errorf("failed to minify translations: %w", err))
	}
}
