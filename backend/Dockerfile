FROM golang:alpine as builder

RUN apk add --no-cache curl git
RUN curl -fsSL -o /bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && chmod +x /bin/dep

# Download and install the latest release of dep
ADD https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

WORKDIR /go/src/github.com/kubeflow/pipelines
COPY . .

# Needed for github.com/mattn/go-sqlite3
RUN apk add --update gcc musl-dev
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
RUN  find . -maxdepth 2 -name "*.py" -exec dsl-compile --py {} --output {}.tar.gz \;

FROM alpine

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
CMD apiserver --config=/config --sampleconfig=/config/sample_config.json
