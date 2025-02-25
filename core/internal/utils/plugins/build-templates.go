//go:build !dev

package pkg

// We don't build template on production
func BuildTemplates(pluginDir string) (err error) {
	return nil
}
