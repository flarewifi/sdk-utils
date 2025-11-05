package sdkutils

// AppConfig is the application configuration.
type AppConfig struct {
	// Examples: en, zh
	Lang string `json:"lang"`

	// Examples: USD, PH, CNY
	Currency string `json:"currency"`

	// Application secret key
	Secret string `json:"secret"`

	// Application channel: development, beta, stable
	Channel string `json:"channel"`
}
