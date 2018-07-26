FROM golang:1.10

# Download and install the latest release of dep
ADD https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

WORKDIR /go/src/github.com/googleprivate/ml/backend
COPY . .

WORKDIR /go/src/github.com/googleprivate/ml/backend/src
RUN dep ensure --vendor-only
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
