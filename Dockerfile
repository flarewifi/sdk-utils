FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    wget tar gzip make gcc git sudo zip

WORKDIR /app

ENV PATH=${PATH}:/home/ubuntu/go/bin
ENV PATH=${PATH}:/usr/local/go/bin

# Install go
COPY .go-version .
RUN wget https://go.dev/dl/go$(cat .go-version).linux-$(dpkg --print-architecture).tar.gz\
        -O golang.tar.gz && \
        rm -rf /usr/local/go && \
        tar -C /usr/local -xzf golang.tar.gz && \
        rm -rf golang.tar.gz

USER ubuntu

# Install core go modules
COPY ./core/go.mod ./core/go.mod
COPY ./core/go.sum ./core/go.sum
RUN cd core && go mod download

# Install additional tools
COPY ./scripts/install-tools.sh .
RUN ./install-tools.sh

# Watch and recompile server on file change
CMD cp go.work.default go.work && \
    reflex \
        -r '\.(go|templ|sql|js|css|json)$' \
        -R '(plugin|package).json$' \
        -R '_templ\.go$' \
        -R '\.tmp\/.*' \
        -R '^output\/.*' \
        -R '^bin\/.*' \
        -R 'db\/queries\/.*' \
        -R 'node_modules' \
        -R 'data\/.*' \
        -R 'resources\/assets\/dist' \
        -R 'storage\/.*' \
        -R 'plugins\/installed\/.*' \
        -R 'plugins\/backups\/.*' \
        -R 'plugins\/updates\/.*' \
        -s -- sh -c './start-dev.sh' -v
