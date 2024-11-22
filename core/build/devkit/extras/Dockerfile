# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.

# Description: Dockerfile for running the devkit.

FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
        wget curl gcc ca-certificates

ENV GOPATH=/build/.tmp/gopath
ENV GOCACHE=/build/.tmp/gocache
ENV GO_CUSTOM_PATH=/build/.tmp/go
ENV PATH=${GO_CUSTOM_PATH}/bin:${PATH}
ENV PATH=${PATH}:/build/.tmp/gopath/bin

WORKDIR /build

CMD cp go.work.default go.work && \
    ./tools.sh && \
    reflex \
        -r '\.go$' \
        -R '\.tmp\/*.' \
        -R 'core\/main\.go' \
        -R 'plugins\/installed' \
        -R 'plugins\/system\/.*\/main\.go' \
        -R 'plugins\/local\/.*\/main\.go' \
        -R 'plugins\/update\/.*\/main\.go' \
        -R 'plugins\/backup\/.*\/main\.go' \
        -R '(.*)mono\.go' \
        -s -- sh -c './start.sh' -v

