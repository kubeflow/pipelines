FROM gcr.io/google.com/cloudsdktool/google-cloud-cli:589.0.0-alpine@sha256:cfbfeb8b32f93d1dd8ab1471c5ed059494f0f0371880e382176931fa05aff831

RUN apk upgrade --no-cache && apk add --no-cache \
    curl \
    jq \
    openssl

RUN gcloud components install kubectl

ADD backend/src/cache/deployer/* /kfp/cache/deployer/
RUN chmod -R 777 /kfp/cache/deployer

WORKDIR /kfp/cache/deployer

RUN chmod +x deploy-cache-service.sh
RUN chmod +x webhook-create-signed-cert.sh
RUN chmod +x webhook-patch-ca-bundle.sh

COPY third_party/license.txt /third_party/license.txt

ENTRYPOINT ["/bin/sh", "/kfp/cache/deployer/deploy-cache-service.sh"]
