FROM golang:1.8

WORKDIR /go/src/github.com/googleprivate/ml/backend
COPY . .

WORKDIR /go/src/github.com/googleprivate/ml/backend/src
RUN go get ./...
# TODO(IronPan): use go dep https://github.com/googleprivate/ml/issues/561
RUN rm -r /go/src/k8s.io/kubernetes/vendor/github.com/golang/glog
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
