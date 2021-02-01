# 1. Build api server application
# Use golang:1.13.1-stretch to keep GLIBC at 2.24 https://github.com/gotify/server/issues/225
FROM golang:1.13.1-stretch as builder
RUN apt-get update && apt-get install -y cmake clang musl-dev openssl
WORKDIR /go/src/github.com/kubeflow/pipelines
COPY . .
RUN go mod vendor
RUN GO111MODULE=on go build -o /bin/apiserver backend/src/apiserver/*.go

# 2. Compile preloaded pipeline samples
FROM python:3.7 as compiler
RUN apt-get update -y && apt-get install --no-install-recommends -y -q default-jdk python3-setuptools python3-dev jq
RUN wget https://bootstrap.pypa.io/get-pip.py && python3 get-pip.py
COPY backend/requirements.txt .
RUN python3 -m pip install -r requirements.txt

# Downloading Argo CLI so that the samples are validated
#ADD  https://github.com/argoproj/argo/releases/download/v2.7.5/argo-linux-amd64 /usr/local/bin/argo
ADD  https://github.com/argoproj/argo/releases/download/v2.4.3/argo-linux-amd64 /usr/local/bin/argo
RUN chmod +x /usr/local/bin/argo

WORKDIR /go/src/github.com/kubeflow/pipelines
COPY sdk sdk
WORKDIR /go/src/github.com/kubeflow/pipelines/sdk/python
RUN python3 setup.py install

WORKDIR /
COPY ./samples /samples
COPY backend/src/apiserver/config/sample_config.json /samples/

# Compiling the preloaded samples.
# The default image is replaced with the GCR-hosted python image.
RUN set -e; \
    < /samples/sample_config.json jq .[].file --raw-output | while read pipeline_yaml; do \
        pipeline_py="${pipeline_yaml%.yaml}"; \
        mv "$pipeline_py" "${pipeline_py}.tmp"; \
        echo 'import kfp; kfp.components.default_base_image_or_builder="gcr.io/google-appengine/python:2020-03-31-141326"' | cat - "${pipeline_py}.tmp" > "$pipeline_py"; \
        python3 "$pipeline_py"; \
    done

# 3. Start api web server
FROM debian:stretch

ARG COMMIT_SHA=unknown
ENV COMMIT_SHA=${COMMIT_SHA}
ARG TAG_NAME=unknown
ENV TAG_NAME=${TAG_NAME}

WORKDIR /bin

COPY third_party/license.txt /bin/license.txt
COPY backend/src/apiserver/config/ /config

COPY --from=builder /bin/apiserver /bin/apiserver
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
CMD /bin/apiserver --config=/config --sampleconfig=/config/sample_config.json -logtostderr=true
