FROM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS builder

RUN apk add --no-cache curl
WORKDIR /src
# Kubernetes v1.37.1, pinned source; retain upstream kubectl behavior while
# rebuilding its embedded standard library and affected Go dependencies.
ARG KUBERNETES_COMMIT=d2b770f4c94636a992534a8d3156b6b2d8e82ac3
ARG KUBERNETES_SHA256=ec99c6a72daa56c2549f42e5d9818b77f288bc7fec3f58acaab5ff604bb38baf
RUN curl -fsSL --retry 3 "https://codeload.github.com/kubernetes/kubernetes/tar.gz/${KUBERNETES_COMMIT}" -o /tmp/kubernetes.tar.gz \
    && echo "${KUBERNETES_SHA256}  /tmp/kubernetes.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/kubernetes.tar.gz --strip-components=1 \
    && rm /tmp/kubernetes.tar.gz
ENV GOWORK=off GOFLAGS=-mod=mod CGO_ENABLED=0
RUN go get golang.org/x/net@v0.60.0 golang.org/x/text@v0.42.0 \
    && go build -trimpath -ldflags="-s -w -X k8s.io/component-base/version.gitVersion=v1.37.1 -X k8s.io/component-base/version.gitMajor=1 -X k8s.io/component-base/version.gitMinor=37 -X k8s.io/component-base/version.gitCommit=${KUBERNETES_COMMIT} -X k8s.io/component-base/version.gitTreeState=modified" -o /out/kubectl ./cmd/kubectl

FROM alpine:3.21

# No runtime scripts use Cloud CLI. Keep only their required tools.
RUN apk upgrade --no-cache && apk add --no-cache \
    bash ca-certificates curl jq openssl
COPY --from=builder /out/kubectl /usr/local/bin/kubectl

ADD backend/src/cache/deployer/* /kfp/cache/deployer/
RUN chmod -R 777 /kfp/cache/deployer

# The deployment runs as UID 1000 and downloads a cluster-matched kubectl here.
ENV HOME=/home/cache-deployer
RUN mkdir -p "${HOME}" && chown 1000:0 "${HOME}" && chmod 0770 "${HOME}"
WORKDIR /kfp/cache/deployer

RUN chmod +x deploy-cache-service.sh webhook-create-signed-cert.sh webhook-patch-ca-bundle.sh

COPY third_party/license.txt /third_party/license.txt

ENTRYPOINT ["/bin/sh", "/kfp/cache/deployer/deploy-cache-service.sh"]
