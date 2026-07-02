// NOTE: Do not use "go mod tidy" to prevent coupling of dependencies.

module sdk/api

go 1.21

toolchain go1.21.13

require (
	github.com/a-h/templ v0.2.793
	github.com/digineo/go-uci v0.0.0-20210918132103-37c7b10c14fa
	github.com/flarewifi/sdk-utils v0.1.15
)

require (
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/ulikunitz/xz v0.5.15 // indirect
)
