@echo off
REM Windows batch equivalent for running a Docker Compose service with arguments

docker compose -f docker-compose.yml -f docker-compose.overrides.yml run -it --rm app ./bin/flare %*

