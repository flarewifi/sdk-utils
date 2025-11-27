.PHONY: default mono postgres create-network openwrt docs-build docs-serve sync-version devkit down deploy-arm64
.PHONY: translations-check check-translations translation-report find-missing create-templates help

default: create-network mono

help:
	@echo "FlareHotspot Makefile Commands"
	@echo "==============================="
	@echo ""
	@echo "Development:"
	@echo "  make                    - Start monolithic build (default)"
	@echo "  make mono               - Start monolithic build with SQLite"
	@echo "  make postgres           - Start plugin-based build with PostgreSQL"
	@echo "  make openwrt            - Start OpenWRT development environment"
	@echo "  make down               - Stop all containers"
	@echo ""
	@echo "Documentation:"
	@echo "  make docs-build         - Build documentation"
	@echo "  make docs-serve         - Serve documentation locally"
	@echo ""
	@echo "Translations:"
	@echo "  make translations-check - Validate all translation files"
	@echo "  make translation-report - Generate detailed translation report"
	@echo "  make find-missing LANG=<code> - Find missing translations for language"
	@echo "  make create-templates LANG=<code> - Create translation templates"
	@echo ""
	@echo "Build & Deploy:"
	@echo "  make sync-version       - Sync version across all components"
	@echo "  make devkit             - Build development kit"
	@echo "  make deploy-arm64       - Deploy to ARM64 device"
	@echo ""
	@echo "Examples:"
	@echo "  make translations-check"
	@echo "  make find-missing LANG=prs"
	@echo "  make create-templates LANG=id"
	@echo ""

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

# Translation management targets
translations-check:
	@go run -tags="dev" ./tools/translator --validate

check-translations: translations-check

translation-report:
	@go run -tags="dev" ./tools/translator --markdown-report translation-report.md
	@echo "Report generated: translation-report.md"

find-missing:
	@if [ -z "$(LANG)" ]; then \
		echo "Usage: make find-missing LANG=<language-code>"; \
		echo "Example: make find-missing LANG=prs"; \
		exit 1; \
	fi
	@go run -tags="dev" ./tools/translator --list-untranslated --language $(LANG)

create-templates:
	@if [ -z "$(LANG)" ]; then \
		echo "Usage: make create-templates LANG=<language-code>"; \
		echo "Example: make create-templates LANG=prs"; \
		exit 1; \
	fi
	@echo "Creating translation templates for language: $(LANG)"
	@go run -tags="dev" ./tools/translator --dry-run --language $(LANG)
	@echo "Note: Use the normal scan (without --dry-run) to actually create files"
