FROM selenium/standalone-chrome:3.141.59-oxygen

USER root

ARG CLOUD_SDK_VERSION=279.0.0
ENV CLOUD_SDK_VERSION=$CLOUD_SDK_VERSION

ARG INSTALL_COMPONENTS=kubectl
ENV PATH "$PATH:/opt/google-cloud-sdk/bin/"
RUN apt-get update -qqy && apt-get install -qqy \
        curl \
        gcc \
        python-dev \
        python-pip \
        apt-transport-https \
        lsb-release \
        openssh-client \
        git \
        gnupg \
    && \
    pip install -U crcmod && \
    export CLOUD_SDK_REPO="cloud-sdk-$(lsb_release -c -s)" && \
    echo "deb https://packages.cloud.google.com/apt $CLOUD_SDK_REPO main" > /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
    apt-get update && apt-get install -y google-cloud-sdk=${CLOUD_SDK_VERSION}-0 $INSTALL_COMPONENTS && \
    rm -rf /var/lib/apt/lists/* /tmp/* /usr/share/locale/* /usr/share/i18n/locales/*


RUN curl --silent --show-error --location https://deb.nodesource.com/setup_8.x | bash - && \
    apt-get install -y nodejs

USER seluser

RUN gcloud config set core/disable_usage_reporting true && \
    gcloud config set component_manager/disable_update_check true
