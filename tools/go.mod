// NOTE: Do not use "go mod tidy" to prevent coupling of dependencies.

module tools

go 1.21

toolchain go1.21.13

require (
	github.com/evanw/esbuild v0.24.0
	github.com/flarehotspot/sdk-utils v0.0.1
	github.com/fsnotify/fsnotify v1.9.0
	github.com/goccy/go-json v0.10.3
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.5.3
)

require (
	github.com/jackc/pgx/v5 v5.7.1 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)
