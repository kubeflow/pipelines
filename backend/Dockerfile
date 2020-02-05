ARG BAZEL_IMAGE=l.gcr.io/google/bazel:0.24.0
FROM $BAZEL_IMAGE as builder

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
RUN apt-get update -y && \
    apt-get install --no-install-recommends -y -q default-jdk python3-setuptools python3-dev
RUN wget https://bootstrap.pypa.io/get-pip.py && python3 get-pip.py
RUN python3 -m pip install tfx==0.21.0rc0

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
RUN  set -e; find core tutorials -maxdepth 2 -name '*.py' -type f | while read pipeline; do python3 "$pipeline"; done

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

# Pin sample doc links to the commit that built the backend image
RUN sed -E "s#/(blob|tree)/master/#/\1/${COMMIT_SHA}/#g" -i /config/sample_config.json

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD apiserver --config=/config --sampleconfig=/config/sample_config.json -logtostderr=true
