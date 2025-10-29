//go:build sqlite

package env

func init() {
	BuildTags = BuildTags + " sqlite"
}
