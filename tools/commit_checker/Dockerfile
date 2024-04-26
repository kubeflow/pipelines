FROM registry.access.redhat.com/ubi8/go-toolset:1.20 as builder

WORKDIR /tmp
RUN git clone https://github.com/openshift/build-machinery-go.git && \
    cd /tmp/build-machinery-go/commitchecker && \
    go build

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.9

WORKDIR /bin

COPY --from=builder /tmp/build-machinery-go/commitchecker/commitchecker /bin/commitchecker
RUN microdnf install git && \
    chmod +x /bin/commitchecker && \
    mkdir -p /src/app-root

WORKDIR /src/app-root
ENTRYPOINT ["commitchecker"]

VOLUME ["/src/app-root"]

LABEL name="Commit Checker tool" \
      summary="commitchecker validates a range of commits in a git repository and ensures they meet specific requirements: \
        1. The author's email address does not start with "root@". \
        2. The message starts with one of: \
            i. UPSTREAM: <PR number|carry|drop>: description \
            ii. UPSTREAM: revert: \
        This is useful for repositories that are downstream forks of upstream repositories."
