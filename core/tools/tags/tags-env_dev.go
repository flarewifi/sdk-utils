//go:build dev || !prod

package tags

func env() string {
	return "dev"
}
