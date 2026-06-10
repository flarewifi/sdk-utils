//go:build !mono

package updates

import (
	"testing"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// TestMetaOwnedMembers verifies the member-hiding rule used to collapse meta
// bundles into a single Software Updates row: a member owned by a bundle is hidden
// from the per-plugin list UNLESS the user also installed it standalone.
func TestMetaOwnedMembers(t *testing.T) {
	cfg := sdkutils.PluginsConfig{
		MetaPlugins: []sdkutils.MetaPlugin{
			{
				Package: "com.acme.bundle",
				Name:    "Acme Bundle",
				Version: "1.0.0",
				Members: []string{"com.acme.alpha", "com.acme.beta"},
			},
		},
		Metadata: []sdkutils.PluginMetadata{
			// alpha is purely a bundle member -> hidden.
			{Package: "com.acme.alpha", Standalone: false},
			// beta is a bundle member the user ALSO installed standalone -> shown.
			{Package: "com.acme.beta", Standalone: true},
			// gamma is a standalone plugin, not in any bundle -> shown.
			{Package: "com.acme.gamma", Standalone: true},
		},
	}

	owned := metaOwnedMembers(cfg)

	if _, hidden := owned["com.acme.alpha"]; !hidden {
		t.Errorf("expected bundle-only member com.acme.alpha to be hidden")
	}
	if _, hidden := owned["com.acme.beta"]; hidden {
		t.Errorf("expected standalone-installed member com.acme.beta to be shown (not hidden)")
	}
	if _, hidden := owned["com.acme.gamma"]; hidden {
		t.Errorf("expected non-member com.acme.gamma to be shown (not hidden)")
	}
	if len(owned) != 1 {
		t.Errorf("expected exactly 1 hidden member, got %d: %v", len(owned), owned)
	}
}
