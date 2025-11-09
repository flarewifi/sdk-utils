FROM ubuntu:24.04

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
    nodejs \
    npm

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

RUN mkdir -p /opt/flarehotspot /var/cache/go && \
    touch /etc/.tkn && \
    chown -R ubuntu:ubuntu \
    /opt/flarehotspot \
    /var/cache/go \
    /etc/.tkn

USER ubuntu

# Install core go modules
COPY ./core/go.mod ./core/go.mod
COPY ./core/go.sum ./core/go.sum
RUN cd core && go mod download

# Install additional tools
COPY ./scripts/install-tools.sh .
RUN ./install-tools.sh

EXPOSE 3000 8000

# Watch and recompile server on file change
CMD [ "./docker-cmd.sh", "./start-dev.sh" ]
