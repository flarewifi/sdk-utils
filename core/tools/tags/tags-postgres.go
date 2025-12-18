//go:build !sqlite || postgres

package tags

func database() string {
	return "postgres"
}
