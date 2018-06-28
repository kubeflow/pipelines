FROM selenium/standalone-chrome

WORKDIR /src

COPY --chown=seluser . .

RUN sudo apt-get update -y && sudo apt-get install -y curl && \
    curl -sL https://deb.nodesource.com/setup_8.x | sudo -E bash - && \
    sudo apt-get install -y nodejs && \
    sudo npm i && sudo npm i -g wait-port && \
    sudo apt-get remove -y curl && \
    sudo rm -rf /var/lib/apt/lists/* /tmp/* /usr/share/locale/* /usr/share/i18n/locales/* && \
    sudo chown -R seluser /src && sudo chmod -R 777 /src

CMD /opt/bin/entry_point.sh & \
    SELENIUM_PID=$! && \
    wait-port 127.0.0.1:4444 -t 20000 && \
    wait-port $ML_PIPELINE_SERVICE_HOST:$ML_PIPELINE_SERVICE_PORT -t 20000 && \
    npm test && \
    EXIT_CODE=$? && \
    pkill $SELENIUM_PID; \
    exit $EXIT_CODE
