# Copyright 2021-2022 The Kubeflow Authors
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

# 1. Build api server application
FROM golang:1.21.7-bookworm as builder
RUN apt-get update && apt-get install -y cmake clang musl-dev openssl
WORKDIR /go/src/github.com/kubeflow/pipelines
COPY . .
RUN GO111MODULE=on go build -o /bin/apiserver backend/src/apiserver/*.go
# Check licenses and comply with license terms.
RUN ./hack/install-go-licenses.sh
# First, make sure there's no forbidden license.
RUN go-licenses check ./backend/src/apiserver
RUN go-licenses csv ./backend/src/apiserver > /tmp/licenses.csv && \
  diff /tmp/licenses.csv backend/third_party_licenses/apiserver.csv && \
  go-licenses save ./backend/src/apiserver --save_path /tmp/NOTICES

# 2. Compile preloaded pipeline samples
FROM python:3.8 as compiler
RUN apt-get update -y && apt-get install --no-install-recommends -y -q default-jdk python3-setuptools python3-dev jq
RUN wget https://bootstrap.pypa.io/get-pip.py && python3 get-pip.py
COPY backend/requirements.txt .
RUN python3 -m pip install -r requirements.txt --no-cache-dir

# Downloading Argo CLI so that the samples are validated
ENV ARGO_VERSION v3.4.17
RUN curl -sLO https://github.com/argoproj/argo-workflows/releases/download/${ARGO_VERSION}/argo-linux-amd64.gz && \
    gunzip argo-linux-amd64.gz && \
    chmod +x argo-linux-amd64 && \
    mv ./argo-linux-amd64 /usr/local/bin/argo

WORKDIR /
COPY ./samples /samples
COPY backend/src/apiserver/config/sample_config.json /samples/

# Compiling the preloaded samples.
# The default image is replaced with the GCR-hosted python image.
RUN set -e; \
    < /samples/sample_config.json jq .[].file --raw-output | while read pipeline_yaml; do \
    pipeline_py="${pipeline_yaml%.yaml}"; \
    python3 "$pipeline_py"; \
    done

# 3. Start api web server
FROM debian:stable

ARG COMMIT_SHA=unknown
ENV COMMIT_SHA=${COMMIT_SHA}
ARG TAG_NAME=unknown
ENV TAG_NAME=${TAG_NAME}
ENV LOG_LEVEL info 

WORKDIR /bin

COPY backend/src/apiserver/config/ /config
COPY --from=builder /bin/apiserver /bin/apiserver

# Copy licenses and notices.
COPY --from=builder /tmp/licenses.csv /third_party/licenses.csv
COPY --from=builder /tmp/NOTICES /third_party/NOTICES
COPY --from=compiler /samples/ /samples/
RUN chmod +x /bin/apiserver

# Adding CA certificate so API server can download pipeline through URL and wget is used for liveness/readiness probe command
RUN apt-get update && apt-get install -y ca-certificates wget

# Pin sample doc links to the commit that built the backend image
RUN sed -E "s#/(blob|tree)/master/#/\1/${COMMIT_SHA}/#g" -i /config/sample_config.json && \
    sed -E "s/%252Fmaster/%252F${COMMIT_SHA}/#g" -i /config/sample_config.json

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD /bin/apiserver --config=/config --sampleconfig=/config/sample_config.json -logtostderr=true --logLevel=${LOG_LEVEL}
