FROM gcr.io/google.com/cloudsdktool/google-cloud-cli:alpine

RUN apk add --update \
    curl \
    jq \
    openssl

RUN gcloud components install kubectl

ADD backend/src/cache/deployer/* /kfp/cache/deployer/
RUN chmod -R 755 /kfp/cache/deployer

WORKDIR /kfp/cache/deployer

RUN chmod +x deploy-cache-service.sh
RUN chmod +x webhook-create-signed-cert.sh
RUN chmod +x webhook-patch-ca-bundle.sh

COPY third_party/license.txt /third_party/license.txt

RUN adduser -S -u 65532 appuser
USER 65532

ENTRYPOINT ["/bin/sh", "/kfp/cache/deployer/deploy-cache-service.sh"]
