.PHONY: default mono create-network openwrt docs-build docs-serve sync-version devkit down deploy-arm64 build-mips deploy-mips
.PHONY: translations-check check-translations translation-report find-missing create-templates help status

default: prep
	docker compose -f docker-compose.yml up app docs sqliteweb \
		--build --remove-orphans --force-recreate

mono: prep
	docker compose -f docker-compose.yml -f docker-compose.mono.yml up app docs sqliteweb \
		--build --remove-orphans --force-recreate

help:
	@echo "FlareHotspot Makefile Commands"
	@echo "==============================="
	@echo ""
	@echo "Development:"
	@echo "  make                    - Start development build (non-mono)"
	@echo "  make mono               - Start mono development build"
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

prep: create-network
	mkdir -p plugins/installed
	cp go.work.default go.work

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
	@# Multi-arch devkit. A CGO -buildmode=plugin core .so can't be cross-compiled,
	@# so each arch is compiled natively in its own buildx platform pass (the
	@# non-native one runs under QEMU — SLOW but correct). All staging (system-plugin
	@# link, sysplugin-prepare, core/go.mod edits, compiles) happens inside the
	@# throwaway builder layer, so the host tree is never mutated. buildx writes one
	@# unzipped tree per platform under output/devkit-stage/; merge-devkit.sh unions
	@# them into a single fat zip that select-arch.sh resolves at container boot.
	@# Prereqs: a docker-container buildx builder + QEMU binfmt (Docker Desktop ships
	@# both; on bare Linux: docker run --privileged --rm tonistiigi/binfmt --install all).
	docker buildx inspect flare-devkit >/dev/null 2>&1 || \
		docker buildx create --name flare-devkit --driver docker-container --bootstrap
	docker buildx build --builder flare-devkit \
		--platform linux/amd64,linux/arm64 \
		-f devkit/Dockerfile.devkit --target export \
		--output type=local,dest=output/devkit-stage .
	./devkit/merge-devkit.sh

restart: down default

down:
	docker compose down

deploy-arm64:
	rm -rf \
		./core/plugin.so \
		./output/mono-bin-files \
		./plugins/installed && \
		GOTOOLCHAIN=go1.21.13 GO_ARCH=arm64 go run -tags="prod" ./core/cmd/create-mono-bin/main.go && \
		rsync -avz --delete --exclude='data' output/mono-bin-files/ root@10.0.0.1:/opt/flarewifi/app/

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
