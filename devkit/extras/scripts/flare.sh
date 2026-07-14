#!/bin/sh

# `compose run` with an explicit command replaces the app service's default CMD,
# so docker-cmd.sh (and its select-arch.sh call) never runs — bin/flare doesn't
# exist yet until an arch is picked. Run select-arch.sh first so this works even
# when the app service was never `up` before. Invoked as `sh ./select-arch.sh`
# (not `./select-arch.sh`) to match docker-cmd.sh — it doesn't depend on the
# executable bit surviving a zip extraction onto Windows/NTFS.
#
# All args are forwarded via `-- "$@"` into the container sh's own positional
# params, then re-expanded with "$@" and exec'd — never string-interpolated —
# so shell metacharacters in an argument (quotes, ;, $, backticks) reach
# bin/flare literally instead of being re-interpreted by the container's shell.
docker compose -f docker-compose.yml run -it --rm app sh -c 'sh ./select-arch.sh && exec ./bin/flare "$@"' -- "$@"
