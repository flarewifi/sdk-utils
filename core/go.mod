// NOTE: Do not use "go mod tidy" to prevent coupling of dependencies.

module core

go 1.25.0

require (
	github.com/Masterminds/semver/v3 v3.5.0
	github.com/a-h/templ v0.3.1020
	github.com/digineo/go-uci v0.0.0-20210918132103-37c7b10c14fa
	github.com/evanw/esbuild v0.28.1
	github.com/flarewifi/sdk-utils v0.1.21
	github.com/fsnotify/fsnotify v1.9.0
	github.com/goccy/go-json v0.10.6
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/csrf v1.7.3
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/mattn/go-sqlite3 v1.14.48
	github.com/shirou/gopsutil/v4 v4.26.6
	github.com/stretchr/testify v1.11.1
	github.com/twitchtv/twirp v8.1.3+incompatible
	github.com/ua-parser/uap-go v0.0.0-20260529044130-17c35e68e58c
	github.com/yuin/goldmark v1.8.4
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.53.0
)

require golang.org/x/crypto v0.54.0

require github.com/ncruces/go-strftime v1.0.0 // indirect

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.10.1 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20260627054121-477a66015f15 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/robfig/cron/v3 v3.0.1
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/tklauser/go-sysconf v0.4.0 // indirect
	github.com/tklauser/numcpus v0.12.0 // indirect
	github.com/ulikunitz/xz v0.5.15 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	lukechampine.com/uint128 v1.3.0 // indirect
	modernc.org/cc/v3 v3.41.0 // indirect
	modernc.org/ccgo/v3 v3.17.0 // indirect
	modernc.org/libc v1.74.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/opt v0.2.0 // indirect
	modernc.org/strutil v1.2.1 // indirect
	modernc.org/token v1.1.0 // indirect
)
