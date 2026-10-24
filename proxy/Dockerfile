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

# Preserve the runtime defaults used by the GCP registration script.
FROM gcr.io/inverting-proxy/agent@sha256:432f609b1e98793723807f62a0a5b900deec491322a73330a9c844210fe70385

# We need --allow-releaseinfo-change, because of https://github.com/kubeflow/pipelines/issues/6311#issuecomment-899224137.
RUN apt-get update --allow-releaseinfo-change && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends curl jq python3-venv \
    && rm -rf /var/lib/apt/lists/*
ENV VIRTUAL_ENV=/opt/proxy-venv
ENV PATH="${VIRTUAL_ENV}/bin:${PATH}"
COPY requirements.txt .
RUN python3 -m venv "${VIRTUAL_ENV}" \
    && "${VIRTUAL_ENV}/bin/python3" -m pip install -r \
    requirements.txt --quiet --no-cache-dir \
    && rm -f requirements.txt \
    && "${VIRTUAL_ENV}/bin/python3" -c "import requests"

# Install gcloud SDK (pinned)
ARG GCLOUD_VERSION=589.0.0
ARG GCLOUD_SHA256=f865da648f6fdae2015abb5b32eaa0eb10520e88f0ca735cfb33795868c8d26f
RUN curl -fsSL "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${GCLOUD_VERSION}-linux-x86_64.tar.gz" > /tmp/google-cloud-sdk.tar.gz
RUN echo "${GCLOUD_SHA256}  /tmp/google-cloud-sdk.tar.gz" | sha256sum -c -
RUN mkdir -p /usr/local/gcloud
RUN tar -C /usr/local/gcloud -xf /tmp/google-cloud-sdk.tar.gz
RUN /usr/local/gcloud/google-cloud-sdk/install.sh --quiet \
    && rm /tmp/google-cloud-sdk.tar.gz
ENV PATH $PATH:/usr/local/gcloud/google-cloud-sdk/bin

# Use the supported Kubernetes minor, with an immutable download checksum.
ARG KUBECTL_VERSION=v1.36.5
ARG KUBECTL_SHA256=33bc88a24c3b09cf55bfd59e6ca977881b61fc3f25ab65bc80c65b430248648e
RUN curl -fsSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" -o /usr/local/bin/kubectl \
    && echo "${KUBECTL_SHA256}  /usr/local/bin/kubectl" | sha256sum -c - \
    && chmod +x /usr/local/bin/kubectl

COPY --from=builder /go/bin/proxy-forwarding-agent /opt/bin/proxy-forwarding-agent

COPY ./ /opt/proxy
# These run in every CI qualification build of the final runtime image.
RUN /opt/bin/proxy-forwarding-agent --help \
    && kubectl version --client=true --output=json \
    && gcloud --version \
    && python3 -m unittest discover -s /opt/proxy -p '*_test.py'

CMD ["/bin/sh", "-c", "/opt/proxy/attempt-register-vm-on-proxy.sh"]
