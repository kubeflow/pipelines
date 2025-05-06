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

FROM registry.access.redhat.com/ubi9/go-toolset:1.22 as builder

USER root
# Downloading Argo CLI so that the samples are validated
ENV ARGO_VERSION v3.4.17
RUN curl -sLO https://github.com/argoproj/argo-workflows/releases/download/${ARGO_VERSION}/argo-linux-amd64.gz && \
    gunzip argo-linux-amd64.gz && \
    chmod +x argo-linux-amd64 && \
    mv ./argo-linux-amd64 /usr/local/bin/argo

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

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5

WORKDIR /bin

COPY --from=builder /opt/app-root/src/backend/src/apiserver/config/ /config
COPY --from=builder /bin/apiserver /bin/apiserver

RUN chmod +x /bin/apiserver

USER root

# Adding CA certificate so API server can download pipeline through URL and wget is used for liveness/readiness probe command
RUN microdnf install -y ca-certificates wget

USER 1001

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD /bin/apiserver --config=/config --sampleconfig=/config/sample_config.json -logtostderr=true --logLevel=${LOG_LEVEL}
