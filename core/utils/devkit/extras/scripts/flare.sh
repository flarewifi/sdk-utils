#!/bin/sh

docker compose -f docker-compose.yml -f docker-compose.overrides.yml run -it --rm app ./bin/flare $1
