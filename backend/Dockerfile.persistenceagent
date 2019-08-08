FROM golang:1.11-alpine3.7 as builder

WORKDIR /go/src/github.com/kubeflow/pipelines
COPY . .

# Needed musl-dev for github.com/mattn/go-sqlite3
RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh gcc musl-dev

RUN GO111MODULE=on go build -o /bin/persistence_agent backend/src/agent/persistence/*.go

FROM alpine:3.8
WORKDIR /bin

COPY --from=builder /bin/persistence_agent /bin/persistence_agent
COPY --from=builder /go/src/github.com/kubeflow/pipelines/third_party/license.txt /bin/license.txt

ENV NAMESPACE ""

CMD persistence_agent --alsologtostderr=true --namespace=${NAMESPACE}
