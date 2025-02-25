//go:build !dev

package plugins

// We don't run sqlc generate on production
func BuildQueries(pluginSrc string) error {
	return nil
}
