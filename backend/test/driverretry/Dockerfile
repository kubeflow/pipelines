# Test-only image. Build driver-retry-test for Linux before building this image.
FROM alpine:3.21
RUN adduser -S -u 65532 appuser
USER 65532
COPY driver-retry-test /bin/driver-retry-test
ENTRYPOINT ["/bin/driver-retry-test"]
