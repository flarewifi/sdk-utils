//go:build !mono

package api

import (
	"fmt"
	"path/filepath"
	"plugin"
	"strings"

	sdkapi "sdk/api"
)

func (api *PluginApi) Init() error {
	pluginLib := filepath.Join(api.dir, "plugin.so")
	p, err := plugin.Open(pluginLib)
	if err != nil {
		// Go's plugin package requires the host and every plugin to be built
		// against identical versions of every shared dependency. A version drift
		// surfaces here as a cryptic "different version of package" error. Wrap it
		// with an actionable message naming the stale plugin.
		if strings.Contains(err.Error(), "different version of package") {
			return fmt.Errorf(
				"plugin %q is stale: it was built against a different dependency version than the core. Rebuild it with `flare build-plugins`. (%w)",
				api.info.Package, err,
			)
		}
		return err
	}

	initSym, err := p.Lookup("Init")
	if err != nil {
		return err
	}

	if initFn, ok := initSym.(func(sdkapi.IPluginApi) error); ok {
		if err := initFn(api); err != nil {
			return err
		}
		return nil
	}

	return nil
}
