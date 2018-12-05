FROM selenium/standalone-chrome

USER root

#TODO: Switch to official GCP Dockerfile method from https://github.com/GoogleCloudPlatform/cloud-sdk-docker/blob/master/debian_slim/Dockerfile
# RUN sudo apt-get install kubectl

# Setup Google Cloud SDK, and use it to install kubectl so it gets the right context
RUN wget -nv https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.zip && \
    unzip -qq google-cloud-sdk.zip -d tools && \
    rm google-cloud-sdk.zip && \
    tools/google-cloud-sdk/install.sh --usage-reporting=false \
    --path-update=false --bash-completion=false \
    --disable-installation-options && \
    tools/google-cloud-sdk/bin/gcloud -q components install kubectl

RUN apt-get update -y && \
    curl --silent --show-error --location https://deb.nodesource.com/setup_8.x | bash - && \
    apt-get install -y nodejs && \
    rm -rf /var/lib/apt/lists/* /tmp/* /usr/share/locale/* /usr/share/i18n/locales/*

USER seluser
