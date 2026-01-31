# Multi-arch Selenium image supporting amd64 and arm64
FROM selenium/standalone-chromium:143.0

USER root
RUN apt-get update -qqy && \
    apt-get install -qqy curl && \
    curl -fsSL https://deb.nodesource.com/setup_18.x | bash - && \
    apt-get install -y nodejs && \
    rm -rf /var/lib/apt/lists/*
USER seluser

COPY --chown=seluser . /src

WORKDIR /src

ENTRYPOINT [ "./run_test.sh" ]
