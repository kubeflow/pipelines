# Use official Python 3.12 slim image
FROM python:3.10-slim

# Install curl to download kubectl
RUN apt-get update && \
    apt-get install -y curl python3-distutils && \
    rm -rf /var/lib/apt/lists/*

# Install kubectl (v1.28.0)
RUN curl -LO "https://dl.k8s.io/release/v1.28.0/bin/linux/amd64/kubectl" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && \
    rm kubectl

# upgrade pip and install pyyaml
RUN python3 -m pip install --upgrade pip && pip install pyyaml
