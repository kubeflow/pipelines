# 0.9.6~0.9.7 has bug and fixed PR447.
FROM gcr.io/cloud-marketplace-tools/k8s/deployer_helm/onbuild:0.9.10
# Install curl
RUN apt-get update -y && apt-get install curl -y
# Install gcloud sdk.
RUN curl https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.tar.gz > /tmp/google-cloud-sdk.tar.gz
RUN mkdir -p /usr/local/gcloud
RUN tar -C /usr/local/gcloud -xf /tmp/google-cloud-sdk.tar.gz
RUN /usr/local/gcloud/google-cloud-sdk/install.sh
ENV PATH $PATH:/usr/local/gcloud/google-cloud-sdk/bin

# move the old entrypoint, because mpdev tool will call it directly, whatever entrypoint is.
RUN mv /bin/deploy.sh /bin/core_deploy.sh
ADD deployer/init_action.sh /bin/deploy.sh
RUN chmod 755 /bin/deploy.sh

ENTRYPOINT ["/bin/bash", "/bin/deploy.sh"]