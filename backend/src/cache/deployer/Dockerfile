FROM google/cloud-sdk:alpine

RUN apk add --update \
    curl \
    which \
    jq \
    bash \
    openssl


RUN curl -LO https://dl.k8s.io/release/v1.21.0/bin/linux/amd64/kubectl
# Double spaces have to be used for alpine between sha256 and file
# more info here: https://github.com/gliderlabs/docker-alpine/issues/174
RUN [ "9f74f2fa7ee32ad07e17211725992248470310ca1988214518806b39b1dad9f0  kubectl" == "$(sha256sum kubectl)" ]
RUN install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

ADD backend/src/cache/deployer/* /kfp/cache/deployer/

WORKDIR /kfp/cache/deployer

RUN chmod +x deploy-cache-service.sh
RUN chmod +x webhook-create-signed-cert.sh
RUN chmod +x webhook-patch-ca-bundle.sh

##COPY third_party/license.txt /bin/license.txt

ENTRYPOINT ["/bin/sh", "/kfp/cache/deployer/deploy-cache-service.sh"]
