FROM alpine

RUN apk add --update \
    curl \
    jq \
    openssl

 RUN ARCH=$(uname -m); \
    if [ "$ARCH" = "x86_64" ]; then \
        ARCH="amd64"; \
    fi \
    && curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/${ARCH}/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin/kubectl

ADD backend/src/cache/deployer/* /kfp/cache/deployer/
RUN chmod -R 777 /kfp/cache/deployer

WORKDIR /kfp/cache/deployer

RUN chmod +x deploy-cache-service.sh
RUN chmod +x webhook-create-signed-cert.sh
RUN chmod +x webhook-patch-ca-bundle.sh

COPY third_party/license.txt /third_party/license.txt

ENTRYPOINT ["/bin/sh", "/kfp/cache/deployer/deploy-cache-service.sh"]
