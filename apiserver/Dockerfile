FROM golang

ARG GOPATH=/go
ARG WEBSERVER_HOME=${GOPATH}/src/github.com/googleprivate/ml/webserver
ARG DBPATH=/go/src/github.com/googleprivate/ml/webserver/db

RUN set -ex \
    && apt-get update -yqq \
    && apt-get install -yqq --no-install-recommends \
        sqlite3 \
        libsqlite3-dev


RUN mkdir -p ${WEBSERVER_HOME}

COPY src/ ${WEBSERVER_HOME}/src
COPY db/ ${WEBSERVER_HOME}/db
COPY config/ ${WEBSERVER_HOME}/config


RUN export GOPATH=${GOPATH} \
    && export PATH=${GOPATH}/bin:$PATH \
    && cd ${WEBSERVER_HOME}/src \
    && go get ./... \
    && go build -o /go/bin/webserver *.go

# Initialize sqlite3 db
RUN sqlite3 {DBPATH}/pipelines.db < {DBPATH}/pipelines.sql

# Expose webserver port
EXPOSE 8888

# Start the webserver
CMD webserver --config=/go/src/github.com/googleprivate/ml/webserver/config/config.json

