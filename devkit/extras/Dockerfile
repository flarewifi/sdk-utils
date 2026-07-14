FROM ubuntu:24.04

# Devkit runtime image. Unlike the in-repo Dockerfile (which COPYs core/go.mod and
# runs `cd core && go mod download`), the devkit ships NO core Go source — the core
# is closed source and present only as the compiled bin/flare + core/plugin.so. So
# this image installs only the toolchain needed to (re)build the developer's own
# plugins at runtime; the extracted devkit tree is bind-mounted onto /app by
# docker-compose.yml.
RUN apt-get update && apt-get install -y \
    wget \
    tar \
    gzip \
    make \
    gcc \
    git \
    sudo \
    zip \
    rsync \
    tree \
    nodejs \
    npm

WORKDIR /app

ENV PATH=${PATH}:/home/ubuntu/go/bin
ENV PATH=${PATH}:/usr/local/go/bin

# Install go (matched to the toolchain the prebuilt binaries were compiled with,
# so plugin .so files are ABI-compatible with core/plugin.so).
COPY .go-version .
RUN wget https://go.dev/dl/go$(cat .go-version).linux-$(dpkg --print-architecture).tar.gz \
        -O golang.tar.gz && \
        rm -rf /usr/local/go && \
        tar -C /usr/local -xzf golang.tar.gz && \
        rm -rf golang.tar.gz

# Seed mount points with ubuntu ownership (Docker seeds empty named volumes from
# the image dir; without this they default to root:root and the app, running as
# ubuntu, cannot write data/storage).
RUN mkdir -p /opt/flarewifi/data/storage /var/cache/go && \
    touch /etc/.tkn && \
    chown -R ubuntu:ubuntu \
    /opt/flarewifi \
    /var/cache/go \
    /etc/.tkn && \
    chown -R ubuntu:ubuntu /app

USER ubuntu

# reflex (file watcher), sqlc + templ (plugin codegen) for the hot-reload loop.
COPY ./scripts/install-tools.sh ./scripts/install-tools.sh
RUN ./scripts/install-tools.sh

EXPOSE 3000 3443 8000

# Watch the developer's plugin sources and (re)build via the prebuilt flare CLI.
CMD [ "./docker-cmd.sh", "./start-dev.sh" ]
