FROM python:3.6-slim
  
# Directories for model codes and secrets
RUN mkdir /app

# Install curl and kubectl
RUN apt-get update
RUN apt-get install -y curl gnupg
RUN apt-get install -y apt-transport-https
RUN curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
RUN echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | tee -a /etc/apt/sources.list.d/kubernetes.list
RUN apt-get update
RUN apt-get install -y kubectl

# Directory for secrets
COPY src/config.py /app
