# Copyright 2019 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Rebuild the pinned agent source with the release's patched Go toolchain.
FROM gcr.io/inverting-proxy/agent@sha256:694d6c1bf299585b530c923c3728cd2c45083f3b396ec83ff799cef1c9dc7474 AS agent-source
FROM golang:1.27.3-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS agent-build
WORKDIR /src
COPY --from=agent-source /opt/src/github.com/google/inverting-proxy/ ./
RUN go get github.com/gorilla/websocket@v1.5.3 \
      golang.org/x/net@v0.60.0 golang.org/x/oauth2@v0.36.0 \
      golang.org/x/sys@v0.48.0 golang.org/x/text@v0.42.0 \
      google.golang.org/grpc@v1.83.2 google.golang.org/api@v0.264.0 \
    && CGO_ENABLED=0 go build -mod=mod -o /opt/bin/proxy-forwarding-agent ./agent/agent.go

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

FROM python:3.11-slim@sha256:9534e5a8e315485d4061ed659af0fd78a284c015f9b73661b41d6bab25604534
RUN apt-get update && apt-get upgrade -y \
    && apt-get install --no-install-recommends -y ca-certificates curl jq wget \
    && rm -rf /var/lib/apt/lists/*
COPY --from=agent-build /opt/bin/proxy-forwarding-agent /opt/bin/proxy-forwarding-agent
COPY requirements.txt /tmp/requirements.txt
RUN python3 -m pip install --quiet --no-cache-dir -r /tmp/requirements.txt \
    && python3 -m pip uninstall -y pip setuptools wheel \
    && rm -rf /usr/local/lib/python3.11/ensurepip \
    && rm /tmp/requirements.txt

# Preserve defaults consumed by attempt-register-vm-on-proxy.sh.
ENV DEBUG=false PROXY_TIMEOUT=60s HEALTH_CHECK_PATH=/ \
    HEALTH_CHECK_INTERVAL_SECONDS=0 HEALTH_CHECK_UNHEALTHY_THRESHOLD=2

# Use the maintained runtime Python instead of a second bundled interpreter.
ARG GCLOUD_VERSION=557.0.0
RUN curl -fsSL "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${GCLOUD_VERSION}-linux-x86_64.tar.gz" -o /tmp/google-cloud-sdk.tar.gz \
    && mkdir -p /usr/local/gcloud \
    && tar -C /usr/local/gcloud -xf /tmp/google-cloud-sdk.tar.gz \
    && rm /tmp/google-cloud-sdk.tar.gz \
    && rm -rf /usr/local/gcloud/google-cloud-sdk/platform/bundledpythonunix \
    && rm /usr/local/gcloud/google-cloud-sdk/bin/gcloud-crc32c
# gcloud-crc32c accelerates Cloud Storage transfers; proxy registration uses
# only gcloud auth, so it does not need this unrelated bundled Go executable.
ENV CLOUDSDK_PYTHON=/usr/local/bin/python3
ENV PATH="${PATH}:/usr/local/gcloud/google-cloud-sdk/bin"

COPY --from=kubectl /out/kubectl /usr/local/bin/kubectl

COPY ./ /opt/proxy
CMD ["/bin/sh", "-c", "/opt/proxy/attempt-register-vm-on-proxy.sh"]
