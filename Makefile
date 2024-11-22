default:
	docker compose up --build --remove-orphans

server-dev:
	./run-dev.sh

openwrt:
	go run ./core/cmd/build-cli/main.go && \
	go run ./core/cmd/build-core/main.go && \
	./bin/flare server

docs-build:
	cd sdk/mkdocs && mkdocs build

docs-serve:
	cd sdk/mkdocs && mkdocs serve

sync-version:
	go run ./core/internal/cli/flare-internal.go sync-version

devkit:
	go run -tags="dev" ./core/internal/cli/flare-internal.go create-devkit
