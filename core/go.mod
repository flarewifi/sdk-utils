// NOTE: Do not use "go mod tidy" to prevent coupling of dependencies.

module core

go 1.21

toolchain go1.21.13

require (
	github.com/a-h/templ v0.2.793
	github.com/digineo/go-uci v0.0.0-20210918132103-37c7b10c14fa
	github.com/evanw/esbuild v0.24.0
	github.com/flarehotspot/go-utils v0.0.4
	github.com/goccy/go-json v0.10.3
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/csrf v1.7.2
	github.com/gorilla/mux v1.8.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/stretchr/testify v1.8.4
	github.com/twitchtv/twirp v8.1.3+incompatible
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/a-h/parse v0.0.0-20240121214402-3caf7543159a // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	golang.org/x/crypto v0.20.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.23.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
