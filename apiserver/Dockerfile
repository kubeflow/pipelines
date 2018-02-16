FROM golang

ARG GOPATH=/go
ARG APISERVER_HOME=${GOPATH}/src/github.com/googleprivate/ml/apiserver
ARG DBPATH=${APISERVER_HOME}/db
ARG BINPATH=${GOPATH}/bin

RUN set -ex \
    && apt-get update -yqq \
    && apt-get install -yqq --no-install-recommends \
        sqlite3 \
        libsqlite3-dev


RUN mkdir -p ${APISERVER_HOME}

COPY src/ ${APISERVER_HOME}/src
COPY db/ ${APISERVER_HOME}/db
COPY config/ ${APISERVER_HOME}/config
COPY entrypoint.sh /script/entrypoint.sh
RUN chmod -R a+rx /script

RUN export GOPATH=${GOPATH} \
    && export PATH=${BINPATH}:$PATH \
    && cd ${APISERVER_HOME}/src \
    && go get ./... \
    && go build -o ${BINPATH}/apiserver *.go

# Initialize sqlite3 db
RUN sqlite3 ${BINPATH}/pipelines.db < ${DBPATH}/pipelines.sql

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD apiserver --config=/go/src/github.com/googleprivate/ml/apiserver/config/config.json