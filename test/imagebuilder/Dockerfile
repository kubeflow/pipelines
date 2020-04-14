# This image contains script to pull github code, build image and push to gcr
# Available at gcr.io/ml-pipeline/image-builder

FROM google/cloud-sdk:279.0.0

COPY ./build.sh /build.sh
RUN chmod +x /build.sh

ENTRYPOINT ["/build.sh"]
