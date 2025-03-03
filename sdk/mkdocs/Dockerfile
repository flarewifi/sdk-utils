FROM python:3.11-alpine3.19 AS build

# Environment variables
ENV PATH="/root/.local/bin:${PATH}"

# Set build directory
WORKDIR /tmp

# Perform build and cleanup artifacts and caches
RUN \
  apk upgrade --update-cache -a \
&& \
  apk add --no-cache \
    cairo \
    freetype-dev \
    git \
    git-fast-import \
    jpeg-dev \
    openssh \
    tini \
    zlib-dev \
&& \
  apk add --no-cache --virtual .build \
    gcc \
    g++ \
    libffi-dev \
    musl-dev \
&& \
  pip install --no-cache-dir --upgrade pip \
&& \
  pip install --no-cache-dir mkdocs mkdocs-material

# Set working directory
WORKDIR /docs

# Expose MkDocs development server port
EXPOSE 8000

# Start development server by default
CMD ["mkdocs", "serve", "--dev-addr=0.0.0.0:8000"]
