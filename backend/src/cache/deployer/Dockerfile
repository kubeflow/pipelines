FROM google/cloud-sdk:alpine

RUN apk add --update \
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

COPY third_party/license.txt /bin/license.txt

ENTRYPOINT ["/bin/sh", "/kfp/cache/deployer/deploy-cache-service.sh"]
