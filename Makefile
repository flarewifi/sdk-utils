default: create-network
	# start docker services in docker-compose.yml
	docker compose up --build --remove-orphans --force-recreate

mono: create-network
	docker compose -f docker-compose.yml -f docker-compose.mono.yml up \
		--build --remove-orphans --force-recreate

create-network:
	# create docker network if not exists
	docker network inspect flare_network >/dev/null 2>&1 || \
		docker network create --driver bridge flare_network

openwrt:
	./start-openwrt-dev.sh

docs-build:
	docker compose run --rm --build docs sh -c \
		'cd /docs && mkdocs build'

docs-serve:
	docker compose up docs

sync-version:
	docker compose run --rm --build app sh -c \
		'go run -tags="prod mono sqlite" ./tools/cmd/sync-versions/main.go'

devkit:
	docker compose -f ./docker-compose.yml \
		-f ./core/build/devkit/extras/docker-compose.override.yml \
		run -it --rm --build app sh -c ./make-devkit.sh

down:
	docker compose down
