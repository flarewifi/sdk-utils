//go:build prod || !dev

package tags

func env() string {
	return "prod"
}
