FROM golang

WORKDIR /go/src/ml/src
COPY . .

RUN go get ./...
RUN go build -o /bin/apiserver apiserver/*.go


FROM ubuntu
RUN set -ex \
    && apt-get update -yqq \
    && apt-get install -yqq --no-install-recommends \
       sqlite3 \
       libsqlite3-dev

WORKDIR /bin

COPY --from=0 /bin/apiserver /bin/apiserver
COPY apiserver/config/ /config

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD apiserver --config=/config
