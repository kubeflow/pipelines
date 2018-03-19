FROM golang

ARG GOPATH=/go
ARG SRC=${GOPATH}/src/ml/src
ARG BINPATH=/bin

RUN set -ex \
    && apt-get update -yqq \
    && apt-get install -yqq --no-install-recommends \
        sqlite3 \
        libsqlite3-dev

RUN mkdir -p ${SRC}

COPY ./ ${SRC}/

ENV GOPATH ${GOPATH}
ENV PATH ${BINPATH}:$PATH
RUN go get ./... \
    && go get github.com/iris-contrib/httpexpect \
    && go get golang.org/x/tools/cmd/goimports

RUN go build -o ${BINPATH}/apiserver ${SRC}/apiserver/*.go

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD apiserver --config=/go/src/ml/src/apiserver/config

