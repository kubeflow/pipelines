# Rebuild kubectl instead of shipping the upstream binary's older Go runtime.
FROM --platform=$BUILDPLATFORM golang:1.27.3-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS kubectl
ARG TARGETARCH
ENV GOWORK=off
WORKDIR /src
RUN curl -fsSL https://github.com/kubernetes/kubernetes/archive/refs/tags/v1.36.1.tar.gz \
      | tar -xz --strip-components=1 \
    && rm -rf vendor \
    && GOFLAGS=-mod=mod go get golang.org/x/net@v0.60.0 \
      golang.org/x/sys@v0.48.0 golang.org/x/text@v0.42.0 \
    && CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -p 1 -mod=mod -ldflags="-X k8s.io/component-base/version.gitVersion=v1.36.1" -o /out/kubectl ./cmd/kubectl \
    && go clean -cache -modcache \
    && rm -rf /src/*

FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

# The deployer needs kubectl, not the Google Cloud SDK or its Python dependencies.
RUN apk upgrade --no-cache && apk add --no-cache \
    bash \
    ca-certificates \
    curl \
    jq \
    openssl

COPY --from=kubectl /out/kubectl /usr/local/bin/kubectl

ADD backend/src/cache/deployer/* /kfp/cache/deployer/
RUN chmod -R 777 /kfp/cache/deployer

WORKDIR /kfp/cache/deployer

RUN chmod +x deploy-cache-service.sh
RUN chmod +x webhook-create-signed-cert.sh
RUN chmod +x webhook-patch-ca-bundle.sh

COPY third_party/license.txt /third_party/license.txt

ENTRYPOINT ["/bin/sh", "/kfp/cache/deployer/deploy-cache-service.sh"]
