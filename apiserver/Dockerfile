FROM golang

ARG GOPATH=/go
ARG APISERVER_HOME=${GOPATH}/src/ml/apiserver
ARG BINPATH=/bin

RUN set -ex \
    && apt-get update -yqq \
    && apt-get install -yqq --no-install-recommends \
        sqlite3 \
        libsqlite3-dev


RUN mkdir -p ${APISERVER_HOME}

COPY src/ ${APISERVER_HOME}/src
COPY config/ ${APISERVER_HOME}/config

ENV GOPATH ${GOPATH}
ENV PATH ${BINPATH}:$PATH
RUN go get ./... \
    && go get github.com/iris-contrib/httpexpect \
    && go get golang.org/x/tools/cmd/goimports

RUN go build -o ${BINPATH}/apiserver ${APISERVER_HOME}/src/*.go

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD apiserver --config=/go/src/ml/apiserver/config

