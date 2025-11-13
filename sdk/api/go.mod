// NOTE: Do not use "go mod tidy" to prevent coupling of dependencies.

module sdk/api

go 1.21

toolchain go1.21.13

require (
	github.com/a-h/templ v0.2.793
	github.com/digineo/go-uci v0.0.0-20210918132103-37c7b10c14fa
	github.com/flarehotspot/sdk-utils v0.1.11
)

require (
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/jackc/pgx/v5 v5.7.1 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)