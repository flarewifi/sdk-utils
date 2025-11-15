#!/bin/sh

docker compose -f docker-compose.yml -f docker-compose.override.yml run -it --rm app go run --tags="dev mono sqlite" ./core/internal/cli/main.go $1
