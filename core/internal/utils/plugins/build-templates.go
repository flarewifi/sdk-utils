//go:build !dev

package plugins

// We don't build template on production
func BuildTemplates(pluginDir string) (err error) {
	return nil
}
