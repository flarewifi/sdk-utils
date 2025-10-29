//go:build mono

package env

func init() {
	BuildTags = BuildTags + " mono"
}
