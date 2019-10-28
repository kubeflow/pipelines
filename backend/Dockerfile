FROM ubuntu:18.04 as builder

RUN apt-get update \
    && apt-get install -y --no-install-recommends software-properties-common \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN apt-get update \
    && apt-get install -y curl \
        openjdk-11-jdk \
        openjdk-11-jre-headless \
        pkg-config \
        zip \
        g++ \
        wget \
        git \
        zlib1g-dev \
        unzip \
        python \
        python-dev \
        python-setuptools \
        python-virtualenv \
        python3.6 \
        python3.6-dev \
        python3-setuptools \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

ENV BAZEL_VERSION=0.24.0

RUN mkdir -p /bazel \
    && cd /bazel \
    && curl -fSsL -O https://github.com/bazelbuild/bazel/releases/download/$BAZEL_VERSION/bazel-$BAZEL_VERSION-dist.zip \
    && unzip bazel-$BAZEL_VERSION-dist.zip \
    && ./compile.sh \
    && cp output/bazel /usr/local/bin \
    && rm -rf /bazel

RUN apt-get update && \
  apt-get install -y cmake clang musl-dev openssl
WORKDIR /go/src/github.com/kubeflow/pipelines

COPY . .

ARG google_application_credentials
ARG use_remote_build=false

# RUN bazel build -c opt --action_env=PATH --define=grpc_no_ares=true backend/src/apiserver:apiserver
RUN if [ "$use_remote_build" = "true" ]; then \
    echo "Using remote build execution ..." && \
    printf "%s" "$google_application_credentials"  > /credentials.json && \
    bazel --bazelrc=tools/bazel_builder/bazelrc \
      build -c opt backend/src/apiserver:apiserver --config=remote \
      --google_credentials=/credentials.json; \
  else \
    echo "Using local build execution..." && \
    bazel --bazelrc=tools/bazel_builder/bazelrc \
      build -c opt backend/src/apiserver:apiserver; \
  fi

# Compile
FROM python:3.5 as compiler

WORKDIR /go/src/github.com/kubeflow/pipelines
COPY sdk sdk
WORKDIR /go/src/github.com/kubeflow/pipelines/sdk/python
RUN python3 setup.py install

WORKDIR /samples
COPY ./samples .

#We need to check that all samples have been compiled without error.
#For find program, the -exec argument is a filter predicate just like -name. It only affects whether the file is "found", not the find's exit code.
#One way to solve this problem is to check whether we have any python pipelines that cannot compile. Here the exit code is the number of such files:
#RUN  bash -e -c 'exit $(find . -maxdepth 2 -name "*.py" ! -exec dsl-compile --py {} --output {}.tar.gz \; -print | wc -l)'
#I think it's better to just use a shell loop though.
#RUN  for pipeline in $(find . -maxdepth 2 -name '*.py' -type f); do dsl-compile --py "$pipeline" --output "$pipeline.tar.gz"; done
#The "for" loop breaks on all whitespace, so we either need to override IFS or use the "read" command instead.
RUN  set -e; find core -maxdepth 2 -name '*.py' -type f | while read pipeline; do dsl-compile --py "$pipeline" --output "$pipeline.tar.gz"; done

FROM debian:stretch

ARG COMMIT_SHA=unknown
ENV COMMIT_SHA=${COMMIT_SHA}

WORKDIR /bin

COPY third_party/license.txt /bin/license.txt
COPY --from=builder /go/src/github.com/kubeflow/pipelines/bazel-bin/backend/src/apiserver/  /usr/local/apiserver
RUN cp /usr/local/apiserver/linux_$(dpkg --print-architecture)_stripped/apiserver /bin/apiserver && \
    rm -rf /usr/local/apiserver
COPY backend/src/apiserver/config/ /config

COPY --from=compiler /samples/ /samples/

# Adding CA certificate so API server can download pipeline through URL
RUN apt-get update && apt-get install -y ca-certificates

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD apiserver --config=/config --sampleconfig=/config/sample_config.json -logtostderr=true
