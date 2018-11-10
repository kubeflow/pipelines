FROM selenium/standalone-chrome

USER root

RUN apt-get update -y && \
    curl --silent --show-error --location https://deb.nodesource.com/setup_8.x | bash - && \
    apt-get install -y nodejs && \
    rm -rf /var/lib/apt/lists/* /tmp/* /usr/share/locale/* /usr/share/i18n/locales/*

USER seluser
