@echo off
REM Windows batch equivalent for running a Docker Compose service with arguments

if "%1"=="" (
    docker compose run -it --rm app ./bin/flare -h
) else (
    docker compose run -it --rm app ./bin/flare %*
)

