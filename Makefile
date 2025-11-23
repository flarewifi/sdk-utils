default: create-network mono

mono: create-network
	docker compose -f docker-compose.yml -f docker-compose.mono.yml up \
		--build --remove-orphans --force-recreate

postgres:
	docker compose up --build --remove-orphans --force-recreate

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
	docker compose run --rm app sh -c \
		'go run -tags="prod mono sqlite" ./tools/cmd/sync-versions/main.go'

devkit:
	docker compose -f ./docker-compose.yml \
		-f ./core/build/devkit/extras/docker-compose.override.yml \
		run -it --rm --build app sh -c ./make-devkit.sh

down:
	docker compose down

deploy-arm64:
	rm -rf \
		./core/plugin.so \
		./output/mono-bin-files \
		./plugins/installed && \
		GO_ARCH=arm64 go run -tags="prod mono sqlite" ./tools/cmd/create-mono-bin/main.go && \
		rsync -avz --delete --exclude='data' output/mono-bin-files/ root@10.0.0.1:/opt/flarehotspot/app/
