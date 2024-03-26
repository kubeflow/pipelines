FROM python:3.7-slim

RUN apt update && \
    apt install -y build-essential && \
    apt install -y curl && \ 
    apt install -y pkg-config && \
    apt-get install -y libssl-dev

RUN ARCH=$(uname -m); \
    if [ "$ARCH" = "x86_64" ]; then \
        ARCH="amd64"; \
    elif [ "$ARCH" = "aarch64" ]; then \
        ARCH="arm64"; \
    elif [ "$ARCH" = "s390x" ]; then \
        ARCH="s390x"; \
    else \
        echo "Unsupported architecture: $ARCH"; exit 1; \
    fi \
    && curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y && \
    /root/.cargo/bin/rustup target add ${ARCH}-unknown-linux-gnu

ENV PKG_CONFIG_PATH=/usr/lib/pkgconfig

ENV PATH="/root/.cargo/bin:${PATH}"
COPY requirements.txt .
RUN python3 -m pip install -r \
    requirements.txt --quiet --no-cache-dir \
    && rm -f requirements.txt

ENV APP_HOME /app
COPY src $APP_HOME
WORKDIR $APP_HOME

ENTRYPOINT ["python"]
CMD ["kfservingdeployer.py"]
