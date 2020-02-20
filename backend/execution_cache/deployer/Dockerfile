FROM alpine:3.8

RUN apk add --update \
 python \
 curl \
 which \
 jq \
 bash \
 openssl

RUN curl -sSL https://sdk.cloud.google.com | bash

ENV PATH $PATH:/root/google-cloud-sdk/bin
RUN gcloud components install kubectl

ADD backend/execution_cache/deployer/* /kfp/execution_cache/deployer/

WORKDIR /kfp/execution_cache/deployer

RUN chmod +x deploy-execution-cache.sh
RUN chmod +x webhook-create-signed-cert.sh
RUN chmod +x webhook-patch-ca-bundle.sh

ENTRYPOINT ["/bin/sh", "/kfp/execution_cache/deployer/deploy-execution-cache.sh"]