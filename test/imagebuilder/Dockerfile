# This image contains script to pull github code, build image and push to gcr
# Available at gcr.io/ml-pipeline/image-builder

# Docker 20.10 retains the legacy builder and negotiates with the Docker 17.10
# daemon used by the image-builder workflow sidecars.
FROM docker:20.10.24-cli@sha256:2967f0819c84dd589ed0a023b9d25dcfe7a3c123d5bf784ffbb77edf55335f0c AS docker-cli

FROM google/cloud-sdk:577.0.0

COPY ./build.sh /build.sh
COPY --from=docker-cli /usr/local/bin/docker /usr/local/bin/docker
RUN chmod +x /build.sh
RUN docker --version

ENTRYPOINT ["/build.sh"]
