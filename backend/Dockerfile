FROM golang

WORKDIR /go/src/github.com/googleprivate/ml/backend
COPY . .

WORKDIR /go/src/github.com/googleprivate/ml/backend/src
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
COPY src/apiserver/config/ /config

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD apiserver --config=/config
