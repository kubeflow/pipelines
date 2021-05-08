FROM google/cloud-sdk:alpine

RUN apk add --update \
    curl \
    which \
    jq \
    bash \
    openssl


RUN curl -LO https://dl.k8s.io/release/v1.21.0/bin/linux/amd64/kubectl
RUN curl -LO https://dl.k8s.io/v1.21.0/bin/linux/amd64/kubectl.sha256
#Double spaces have to be used for alpine between sha256 and file
#more info here: https://github.com/gliderlabs/docker-alpine/issues/174
RUN echo "$(<kubectl.sha256)  kubectl" | sha256sum -c
RUN install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

ADD backend/src/cache/deployer/* /kfp/cache/deployer/

WORKDIR /kfp/cache/deployer

RUN chmod +x deploy-cache-service.sh
RUN chmod +x webhook-create-signed-cert.sh
RUN chmod +x webhook-patch-ca-bundle.sh

COPY third_party/license.txt /bin/license.txt

ENTRYPOINT ["/bin/sh", "/kfp/cache/deployer/deploy-cache-service.sh"]
