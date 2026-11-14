FROM golang:1.27.4-alpine@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS builder

RUN apk add --no-cache git
WORKDIR /src
# Keep real tag metadata so Go identifies the module version, not (devel).
# Patch Go dependencies for security fixes; the go.mod changes yield +dirty.
ARG KUBERNETES_VERSION=v1.37.1
ARG KUBERNETES_TAG_OBJECT=d2b770f4c94636a992534a8d3156b6b2d8e82ac3
ARG KUBERNETES_COMMIT=f78e722310e50bcaca9276be22276d9e91d91308
RUN git clone --depth=1 --branch "${KUBERNETES_VERSION}" https://github.com/kubernetes/kubernetes.git . \
    && test "$(git rev-parse "refs/tags/${KUBERNETES_VERSION}")" = "${KUBERNETES_TAG_OBJECT}" \
    && test "$(git rev-parse HEAD)" = "${KUBERNETES_COMMIT}"
ENV GOWORK=off GOFLAGS=-mod=mod CGO_ENABLED=0
RUN kubernetes_version="${KUBERNETES_VERSION#v}" \
    && kubernetes_major="${kubernetes_version%%.*}" \
    && kubernetes_minor="${kubernetes_version#*.}" \
    && kubernetes_minor="${kubernetes_minor%%.*}" \
    && go get golang.org/x/net@v0.62.0 golang.org/x/crypto@v0.59.0 golang.org/x/text@v0.44.0 \
      go.opentelemetry.io/otel@v1.47.0 \
      go.opentelemetry.io/otel/trace@v1.47.0 \
      go.opentelemetry.io/otel/metric@v1.47.0 \
      go.opentelemetry.io/otel/log@v1.47.0 \
      go.opentelemetry.io/otel/sdk@v1.47.0 \
      go.opentelemetry.io/otel/sdk/metric@v1.47.0 \
      go.opentelemetry.io/otel/exporters/stdout/stdouttrace@v1.47.0 \
      go.opentelemetry.io/otel/exporters/otlp/otlptrace@v1.47.0 \
      go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.47.0 \
    && go build -trimpath -buildvcs=true -ldflags="-s -w -X k8s.io/component-base/version.gitVersion=${KUBERNETES_VERSION} -X k8s.io/component-base/version.gitMajor=${kubernetes_major} -X k8s.io/component-base/version.gitMinor=${kubernetes_minor} -X k8s.io/component-base/version.gitCommit=${KUBERNETES_COMMIT} -X k8s.io/component-base/version.gitTreeState=modified" -o /out/kubectl ./cmd/kubectl \
    && go version -m /out/kubectl > /out/kubectl.buildinfo \
    && cat /out/kubectl.buildinfo \
    && awk -v expected="${KUBERNETES_VERSION}+dirty" -v commit="${KUBERNETES_COMMIT}" \
      '$1 == "mod" && $2 == "k8s.io/kubernetes" { version = $3 } \
       $1 == "build" && $2 == "vcs.revision=" commit { revision = 1 } \
       END { exit !(version == expected && revision) }' /out/kubectl.buildinfo \
    && go list -m all > /out-modules.txt \
    && awk '$1 == "go.opentelemetry.io/otel/metric/x" { experimental++; if ($2 != "v0.69.0") bad = 1; next } \
      $1 ~ /^go.opentelemetry.io\/otel(\/|$)/ { stable++; if ($2 != "v1.47.0") bad = 1 } \
      END { exit (bad || stable != 9 || experimental != 1) }' /out-modules.txt

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
