# Dockerfile for building the source code of execution_cache
FROM golang:1.11-alpine3.7 as builder

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh gcc musl-dev

ADD backend/execution_cache/* /kfp/execution_cache/

WORKDIR /kfp/execution_cache/

RUN GO111MODULE=on go build -o /bin/execution_cache /kfp/execution_cache/*.go

FROM alpine:3.8
WORKDIR /bin

COPY --from=builder /bin/execution_cache /bin/execution_cache
RUN mkdir /usr/license
COPY --from=builder /kfp/execution_cache/license.txt /usr/license

ENTRYPOINT [ "/bin/execution_cache" ]