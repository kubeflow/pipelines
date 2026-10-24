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

# Build the upstream agent from immutable source with patched, locked dependencies.
FROM golang:1.27.3-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS builder
ARG AGENT_COMMIT=6264998ae706281fd068e8f3eeb3cbd42046c5b6
ARG AGENT_SHA256=d915c04e75836b5a31dda457ff8ad9d235230e4e483f1d2fdb6d42154fb5ea3a
WORKDIR /src
RUN curl -fsSL "https://github.com/google/inverting-proxy/archive/${AGENT_COMMIT}.tar.gz" -o /tmp/agent.tar.gz \
    && echo "${AGENT_SHA256}  /tmp/agent.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/agent.tar.gz --strip-components=1 -C /src
COPY agent.go.mod go.mod
COPY agent.go.sum go.sum
ENV GOTOOLCHAIN=local
RUN CGO_ENABLED=0 go build -mod=readonly -trimpath -o /go/bin/proxy-forwarding-agent ./agent \
    && CGO_ENABLED=0 go build -mod=readonly -trimpath -o /go/bin/inverting-proxy ./server \
    && go test -mod=readonly -count=1 ./agent/...

# Match the 2.17.3 rebuild pattern while retaining the newer kubectl patch.
# Reuse the canonical compiler pin and keep Kubernetes separate from agent source.
FROM builder AS kubectl
ARG TARGETARCH
ARG KUBECTL_VERSION=v1.36.5
ARG KUBECTL_COMMIT=ad950d1cc78b0183c476bd4d3f1934c104229727
ENV GOWORK=off
WORKDIR /src/kubernetes
# Real tag metadata lets Go report the module version; patches retain +dirty.
RUN git clone --depth=1 --branch "${KUBECTL_VERSION}" https://github.com/kubernetes/kubernetes.git . \
    && test "$(git rev-parse HEAD)" = "${KUBECTL_COMMIT}" \
    && rm -rf vendor \
    && GOFLAGS=-mod=mod go get golang.org/x/net@v0.60.0 \
      golang.org/x/sys@v0.48.0 golang.org/x/text@v0.42.0 \
    && CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -p 1 -mod=mod -buildvcs=true \
      -ldflags="-X k8s.io/component-base/version.gitVersion=${KUBECTL_VERSION}" \
      -o /out/kubectl ./cmd/kubectl \
    && go version -m /out/kubectl > /out/kubectl.buildinfo \
    && cat /out/kubectl.buildinfo \
    && awk -v expected="${KUBECTL_VERSION}+dirty" -v commit="${KUBECTL_COMMIT}" \
      '$1 == "mod" && $2 == "k8s.io/kubernetes" { version = $3 } \
       $1 == "build" && $2 == "vcs.revision=" commit { revision = 1 } \
       END { exit !(version == expected && revision) }' /out/kubectl.buildinfo

# Keep only the runtime tools needed by registration and the Cloud SDK.
FROM debian:trixie-slim@sha256:62bd423aeee912639ca8ee681c2b762cb3e25137e0a92b20be455898a9f034ad
RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends bash ca-certificates curl jq python3 python3-venv \
    && rm -rf /var/lib/apt/lists/*

# These defaults were previously inherited from the upstream agent image.
ENV DEBUG=false \
    PROXY_TIMEOUT=60s \
    HEALTH_CHECK_PATH=/ \
    HEALTH_CHECK_INTERVAL_SECONDS=0 \
    HEALTH_CHECK_UNHEALTHY_THRESHOLD=2
WORKDIR /opt/proxy
ENV VIRTUAL_ENV=/opt/proxy-venv
ENV PATH="${VIRTUAL_ENV}/bin:${PATH}"
# Keep the interpreter explicitly installed; venv creation and pip are build tools.
COPY requirements.txt .
RUN python3 -m venv "${VIRTUAL_ENV}" \
    && "${VIRTUAL_ENV}/bin/python3" -m pip install -r \
    requirements.txt --quiet --no-cache-dir \
    && rm -f requirements.txt \
    && "${VIRTUAL_ENV}/bin/python3" -c "import requests" \
    && "${VIRTUAL_ENV}/bin/python3" -m pip uninstall -y pip \
    && apt-get purge -y python3-venv python3.13-venv python3-pip-whl python3-setuptools-whl

# Install gcloud SDK (pinned)
ARG GCLOUD_VERSION=589.0.0
ARG GCLOUD_SHA256=f865da648f6fdae2015abb5b32eaa0eb10520e88f0ca735cfb33795868c8d26f
RUN curl -fsSL "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${GCLOUD_VERSION}-linux-x86_64.tar.gz" > /tmp/google-cloud-sdk.tar.gz
RUN echo "${GCLOUD_SHA256}  /tmp/google-cloud-sdk.tar.gz" | sha256sum -c -
RUN mkdir -p /usr/local/gcloud
RUN tar -C /usr/local/gcloud -xf /tmp/google-cloud-sdk.tar.gz
# As in 2.17.3, registration uses gcloud auth, not Cloud Storage transfers.
# Use the maintained runtime Python instead of a second bundled interpreter.
RUN rm -rf /usr/local/gcloud/google-cloud-sdk/platform/bundledpythonunix \
    && rm /usr/local/gcloud/google-cloud-sdk/bin/gcloud-crc32c \
    && rm /tmp/google-cloud-sdk.tar.gz
ENV CLOUDSDK_PYTHON=/usr/bin/python3
ENV PATH $PATH:/usr/local/gcloud/google-cloud-sdk/bin

COPY --from=kubectl /out/kubectl /usr/local/bin/kubectl

COPY --from=builder /go/bin/proxy-forwarding-agent /opt/bin/proxy-forwarding-agent
COPY --from=builder /src/LICENSE /usr/share/doc/inverting-proxy/LICENSE

COPY attempt-register-vm-on-proxy.sh get_proxy_url.py get_proxy_url_test.py registration_test.py /opt/proxy/
COPY security /tmp/proxy-security
# These run in every CI qualification build of the final runtime image.
RUN /opt/bin/proxy-forwarding-agent --help \
    && kubectl version --client=true --output=json \
    && gcloud --version \
    && test "$(gcloud info --format='value(basic.python_location)')" = "${CLOUDSDK_PYTHON}" \
    && gcloud auth activate-service-account --help >/dev/null \
    && gcloud auth print-access-token --help >/dev/null \
    && test ! -e /usr/local/gcloud/google-cloud-sdk/platform/bundledpythonunix \
    && test ! -e /usr/local/gcloud/google-cloud-sdk/bin/gcloud-crc32c \
    && python3 -c 'import importlib.util; import requests; assert importlib.util.find_spec("pip") is None' \
    && python3 -m unittest discover -s /opt/proxy -p '*_test.py' \
    && python3 -m unittest discover -s /tmp/proxy-security -p '*_test.py' \
    && rm -rf /tmp/proxy-security /opt/proxy/get_proxy_url_test.py /opt/proxy/registration_test.py

CMD ["/bin/sh", "-c", "/opt/proxy/attempt-register-vm-on-proxy.sh"]
