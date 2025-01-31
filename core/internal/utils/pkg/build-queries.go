//go:build !dev

package pkg

// We don't run sqlc generate on production
func BuildQueries(pluginSrc string) error {
	return nil
}
