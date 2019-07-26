FROM selenium/standalone-chrome:3.141.59-oxygen

USER root

WORKDIR /src

COPY . .

RUN apt-get update -y && apt-get install -y curl && \
    curl -sL https://deb.nodesource.com/setup_8.x | bash - && \
    apt-get install -y nodejs && \
    apt-get remove -y curl && \
    rm -rf /var/lib/apt/lists/* /tmp/* /usr/share/locale/* /usr/share/i18n/locales/* && \
    chown -R seluser /src && chmod -R 777 /src

USER seluser

ENTRYPOINT [ "./run_test.sh" ]
