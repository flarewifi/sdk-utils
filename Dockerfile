FROM ubuntu:24.04

RUN apt-get update && \
        apt-get install -y \
        wget curl gcc golang-go git ca-certificates

ENV TEMP_PATH=/var/tmp/flare.tmp
ENV GOPATH=${TEMP_PATH}/gopath
ENV GOCACHE=${TEMP_PATH}/gocache
ENV GO_CUSTOM_PATH=${TEMP_PATH}/go
ENV PATH=${GO_CUSTOM_PATH}/bin:${PATH}
ENV PATH=${PATH}:${TEMP_PATH}/gopath/bin

WORKDIR /build

CMD cp go.work.default go.work && \
    go run --tags=dev ./core/internal/cli/main.go install-go && \
    go run --tags=dev ./core/cmd/sync-versions/main.go && \
    ./tools.sh && \
    reflex \
        -r '\.(go|templ|js|css|json)$' \
        -R 'assets\/dist\/.*' \
        -R 'db/sqlc/.*' \
        -R 'config\/.*' \
        -R 'node_modules' \
        -R '_templ\.go$' \
        -R 'core\/main\.go' \
        -R 'plugins\/system\/.*\/main\.go$' \
        -R 'plugins\/local\/.*\/main\.go$' \
        -R 'plugins\/installed\/.*' \
        -R 'plugins\/update\/.*' \
        -R 'plugins\/backup\/.*' \
        -R '(.*)mono\.go' \
        -R '\.tmp\/*.' \
        -s -- sh -c './start.sh' -v
