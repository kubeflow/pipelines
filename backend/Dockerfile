FROM golang:alpine as builder

RUN apk add --no-cache curl git
RUN curl -fsSL -o /bin/dep https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 && chmod +x /bin/dep

# Download and install the latest release of dep
ADD https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

WORKDIR /go/src/github.com/googleprivate/ml/backend
COPY . .

WORKDIR /go/src/github.com/googleprivate/ml/backend/src
RUN /bin/dep ensure --vendor-only

# Needed for github.com/mattn/go-sqlite3
RUN apk add --update gcc musl-dev
RUN go build -o /bin/apiserver apiserver/*.go

FROM alpine

ARG COMMIT_SHA=unknown
ENV COMMIT_SHA=${COMMIT_SHA}

WORKDIR /bin

COPY --from=builder /bin/apiserver /bin/apiserver
COPY src/apiserver/config/ /config

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD apiserver --config=/config
