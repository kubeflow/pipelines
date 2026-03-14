# Multi-arch Selenium image supporting amd64 and arm64
FROM selenium/standalone-chromium:143.0

# Derive from frontend/.nvmrc at build time:
#   docker build --build-arg NODE_VERSION=$(tr -d 'v' < ../../frontend/.nvmrc) .
ARG NODE_VERSION=24.14.0

USER root
RUN apt-get update -qqy && \
    apt-get install -qqy curl xz-utils && \
    ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "amd64" ]; then NODE_ARCH=x64; else NODE_ARCH=$ARCH; fi && \
    curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${NODE_ARCH}.tar.xz" \
      | tar -xJ -C /usr/local --strip-components=1 && \
    rm -rf /var/lib/apt/lists/*
USER seluser

COPY --chown=seluser . /src

WORKDIR /src

ENTRYPOINT [ "./run_test.sh" ]
