FROM golang:1.10.2-alpine3.7 AS builder

RUN apk add --no-cache git
RUN go get github.com/golang/dep/cmd/dep

WORKDIR /go/src/github.com/kubeflow/pipelines/
COPY . .

RUN dep ensure -vendor-only -v
RUN go build -o /bin/controller ./resources/scheduledworkflow/*.go

FROM alpine:3.7
COPY --from=builder /bin/controller /bin/controller

CMD ["/bin/controller"]
