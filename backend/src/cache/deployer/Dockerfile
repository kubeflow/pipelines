FROM google/cloud-sdk:alpine

RUN apk add --update \
 python \
 curl \
 which \
 jq \
 bash \
 openssl

RUN gcloud components install kubectl

ADD backend/src/cache/deployer/* /kfp/cache/deployer/

WORKDIR /kfp/cache/deployer

RUN chmod +x deploy-cache-service.sh
RUN chmod +x webhook-create-signed-cert.sh
RUN chmod +x webhook-patch-ca-bundle.sh

ENTRYPOINT ["/bin/sh", "/kfp/cache/deployer/deploy-cache-service.sh"]