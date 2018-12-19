FROM golang:alpine as builder

WORKDIR /go/src/github.com/kubeflow/pipelines
COPY . .

RUN apk add --no-cache gcc musl-dev
RUN go build -o /bin/apiserver backend/src/apiserver/*.go

FROM python:3.5 as compiler

RUN apt-get update -y && \
    apt-get install --no-install-recommends -y -q default-jdk wget

RUN pip3 install setuptools==40.5.0

RUN wget http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.3.1/swagger-codegen-cli-2.3.1.jar -O /tmp/swagger-codegen-cli.jar

WORKDIR /go/src/github.com/kubeflow/pipelines
COPY . .
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


FROM alpine:3.8

ARG COMMIT_SHA=unknown
ENV COMMIT_SHA=${COMMIT_SHA}

WORKDIR /bin

COPY --from=builder /bin/apiserver /bin/apiserver
COPY --from=builder /go/src/github.com/kubeflow/pipelines/third_party/license.txt /bin/license.txt
COPY backend/src/apiserver/config/ /config

COPY --from=compiler /samples/ /samples/

# Adding CA certificate so API server can download pipeline through URL
RUN apk add ca-certificates

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD apiserver --config=/config
