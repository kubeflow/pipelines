FROM l.gcr.io/google/bazel:0.21.0 as builder

RUN apt-get update && \
  apt-get install -y cmake clang musl-dev openssl
WORKDIR /go/src/github.com/kubeflow/pipelines

COPY WORKSPACE WORKSPACE
COPY backend/src backend/src
COPY backend/api backend/api

RUN bazel build -c opt --action_env=PATH --define=grpc_no_ares=true backend/src/apiserver:apiserver


# Compile
FROM python:3.5 as compiler

RUN apt-get update -y && \
  apt-get install --no-install-recommends -y -q default-jdk wget
RUN pip3 install setuptools==40.5.0

RUN wget http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.1/swagger-codegen-cli-2.4.1.jar -O /tmp/swagger-codegen-cli.jar

# WORKDIR /go/src/github.com/kubeflow/pipelines
WORKDIR /go/src/github.com/kubeflow/pipelines
COPY backend/api  backend/api
COPY sdk sdk
WORKDIR /go/src/github.com/kubeflow/pipelines/sdk/python
RUN ./build.sh /kfp.tar.gz
RUN pip3 install /kfp.tar.gz

WORKDIR /samples
COPY ./samples .

#We need to check that all samples have been compiled without error.
#For find program, the -exec argument is a filter predicate just like -name. It only affects whether the file is "found", not the find's exit code.
#One way to solve this problem is to check whether we have any python pipelines that cannot compile. Here the exit code is the number of such files:
#RUN  bash -e -c 'exit $(find . -maxdepth 2 -name "*.py" ! -exec dsl-compile --py {} --output {}.tar.gz \; -print | wc -l)'
#I think it's better to just use a shell loop though.
#RUN  for pipeline in $(find . -maxdepth 2 -name '*.py' -type f); do dsl-compile --py "$pipeline" --output "$pipeline.tar.gz"; done
#The "for" loop breaks on all whitespace, so we either need to override IFS or use the "read" command instead.
RUN  find . -maxdepth 2 -name '*.py' -type f | while read pipeline; do dsl-compile --py "$pipeline" --output "$pipeline.tar.gz"; done

FROM debian:stretch

ARG COMMIT_SHA=unknown
ENV COMMIT_SHA=${COMMIT_SHA}

WORKDIR /bin

COPY third_party/license.txt /bin/license.txt
COPY --from=builder /go/src/github.com/kubeflow/pipelines/bazel-bin/backend/src/apiserver/linux_amd64_stripped/apiserver  /bin/apiserver
COPY backend/src/apiserver/config/ /config

COPY --from=compiler /samples/ /samples/

# Adding CA certificate so API server can download pipeline through URL
RUN apt-get update && apt-get install -y ca-certificates

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD apiserver --config=/config --sampleconfig=/config/sample_config.json -alsologtostderr=true
