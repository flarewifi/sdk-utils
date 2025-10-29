//go:build !sqlite || postgres

package env

func init() {
	BuildTags = BuildTags + " postgres"
}
