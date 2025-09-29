default:
	# create docker network if not exists
	docker network inspect flare_network >/dev/null 2>&1 || \
		docker network create --driver bridge flare_network
	# start docker services in docker-compose.yml
	docker compose up --build --remove-orphans --force-recreate

server-dev:
	./run-dev.sh

openwrt:
	go run ./core/cmd/build-cli/main.go && \
	go run ./core/cmd/build-core/main.go && \
	./bin/flare server

docs-build:
	docker compose run --rm --build docs sh -c 'cd /docs && mkdocs build'

docs-serve:
	docker compose up docs

sync-version:
	go run ./core/internal/cli/flare-internal.go sync-version

devkit:
	docker compose run -it --rm --build app sh -c 'go run --tags=dev ./core/cmd/create-devkit/main.go'
