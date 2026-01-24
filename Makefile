.PHONY: default create-network openwrt docs-build docs-serve sync-version devkit down deploy-arm64 build-mips deploy-mips
.PHONY: translations-check check-translations translation-report find-missing create-templates help status

default: create-network
	docker compose -f docker-compose.yml -f docker-compose.mono.yml up app docs sqliteweb \
		--build --remove-orphans --force-recreate

help:
	@echo "FlareHotspot Makefile Commands"
	@echo "==============================="
	@echo ""
	@echo "Development:"
	@echo "  make                    - Start development build (default)"
	@echo "  make restart            - Stop all containers, then restart"
	@echo "  make openwrt            - Start OpenWRT development environment"
	@echo "  make down               - Stop all containers"
	@echo "  make status             - Show git status for core and plugins"
	@echo ""
	@echo "Documentation:"
	@echo "  make docs-build         - Build documentation"
	@echo "  make docs-serve         - Serve documentation locally"
	@echo ""
	@echo "Build & Deploy:"
	@echo "  make sync-version       - Sync version across all components"
	@echo "  make devkit             - Build development kit"
	@echo "  make deploy-arm64       - Deploy to ARM64 device"
	@echo ""

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
		'go run -tags="prod" ./core/cmd/sync-versions/main.go'

devkit:
	docker compose -f ./docker-compose.yml \
		-f ./core/build/devkit/extras/docker-compose.override.yml \
		run -it --rm --build app sh -c ./make-devkit.sh

restart: down default

down:
	docker compose down

deploy-arm64:
	rm -rf \
		./core/plugin.so \
		./output/mono-bin-files \
		./plugins/installed && \
		GO_ARCH=arm64 go run -tags="prod" ./core/cmd/create-mono-bin/main.go && \
		rsync -avz --delete --exclude='data' output/mono-bin-files/ root@10.0.0.1:/opt/flarehotspot/app/

translate-help:
	@go run core/tools/translator/main.go --help

translate-check:
	@go run -tags="dev" ./core/tools/translator \
		--validate \
		--markdown-report=.tmp/reports/translation_validation_report.md

# Language-specific translation checks
translate-check-%:
	@go run -tags="dev" ./core/tools/translator \
		--language=$* \
		--validate \
		--markdown-report=.tmp/reports/translation_validation_$*_report.md

status:
	@./scripts/status.sh
