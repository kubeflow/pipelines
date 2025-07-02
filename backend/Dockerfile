# Copyright 2021-2024 The Kubeflow Authors
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

# Build arguments
ARG SOURCE_CODE=.

FROM registry.access.redhat.com/ubi9/go-toolset:1.23 AS builder

USER root

RUN dnf install -y cmake clang openssl

COPY ${SOURCE_CODE}/go.mod ./
COPY ${SOURCE_CODE}/go.sum ./
COPY ${SOURCE_CODE}/kubernetes_platform/go.mod ./kubernetes_platform/go.mod
COPY ${SOURCE_CODE}/api/go.mod ./api/go.mod

RUN GO111MODULE=on go mod download

# Copy the source
COPY ${SOURCE_CODE}/ ./

RUN GO111MODULE=on CGO_ENABLED=1 GOEXPERIMENT=strictfipsruntime go build -tags strictfipsruntime -o /bin/apiserver ./backend/src/apiserver/ && \
    dnf clean all

# 2. Compile preloaded pipeline samples
FROM registry.access.redhat.com/ubi9/python-39:9.5 AS compiler

ARG TARGETOS TARGETARCH

USER root
RUN dnf install -y python3-setuptools jq
RUN wget https://bootstrap.pypa.io/get-pip.py && python3 get-pip.py
COPY backend/requirements.txt .
RUN python3 -m pip install -r requirements.txt --no-cache-dir

# Downloading Argo CLI so that the samples are validated
ENV ARGO_VERSION=v3.5.14
RUN curl -sLO https://github.com/argoproj/argo-workflows/releases/download/${ARGO_VERSION}/argo-${TARGETOS:-linux}-${TARGETARCH:-amd64}.gz && \
    gunzip argo-${TARGETOS:-linux}-${TARGETARCH:-amd64}.gz && \
    chmod +x argo-${TARGETOS:-linux}-${TARGETARCH:-amd64} && \
    mv ./argo-${TARGETOS:-linux}-${TARGETARCH:-amd64} /usr/local/bin/argo

WORKDIR /
COPY ./samples /samples
COPY backend/src/apiserver/config/sample_config.json /samples/

# Compiling the preloaded samples.
# The default image is replaced with the GCR-hosted python image.
RUN set -e; \
    < /samples/sample_config.json jq ".pipelines[].file" --raw-output | while read pipeline_yaml; do \
    pipeline_py="${pipeline_yaml%.yaml}"; \
    echo "Compiling: \"$pipeline_py\"" && python3 "$pipeline_py" && echo -n "Output: " && ls "$pipeline_py.yaml"; \
    done

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5

WORKDIR /bin

COPY --from=builder /opt/app-root/src/backend/src/apiserver/config/ /config
COPY --from=builder /bin/apiserver /bin/apiserver

COPY --from=compiler /samples/ /samples/
RUN chmod +x /bin/apiserver

USER root

# Adding CA certificate so API server can download pipeline through URL and wget is used for liveness/readiness probe command
RUN microdnf install -y ca-certificates wget

USER 1001

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD /bin/apiserver --config=/config --sampleconfig=/config/sample_config.json -logtostderr=true --logLevel=${LOG_LEVEL}
