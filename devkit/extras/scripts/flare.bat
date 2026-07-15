@echo off
REM Windows batch equivalent for running a Docker Compose service with arguments
REM `compose run` with an explicit command replaces the app service's default CMD,
REM so docker-cmd.sh (and its select-arch.sh call) never runs — bin/flare doesn't
REM exist yet until an arch is picked. Run select-arch.sh first so this works even
REM when the app service was never `up` before. Invoked as `sh ./select-arch.sh`
REM (not `./select-arch.sh`) to match docker-cmd.sh — it doesn't depend on the
REM executable bit surviving a zip extraction onto Windows/NTFS.
REM
REM All args are forwarded via `-- %*` into the container sh's own positional
REM params, then re-expanded with "$@" and exec'd — never string-interpolated —
REM so shell metacharacters in an argument reach bin/flare literally instead of
REM being re-interpreted by the container's shell.

docker compose -f docker-compose.yml run -it --rm app sh -c "sh ./select-arch.sh && exec ./bin/flare \"$@\"" -- %*

