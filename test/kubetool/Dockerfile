# This is a dockerfile for an image with kubectl and ksonnet installed
# Available at gcr.io/ml-pipeline/kubetool

FROM ubuntu

RUN apt-get update -y
RUN apt-get install -y curl

# Install kubectl
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv kubectl /usr/local/bin/

# Install ksonnet
RUN curl -LO https://github.com/ksonnet/ksonnet/releases/download/v0.9.2/ks_0.9.2_linux_amd64.tar.gz
RUN tar -xzf ks_0.9.2_linux_amd64.tar.gz
RUN chmod +x ./ks_0.9.2_linux_amd64/ks
RUN mv ./ks_0.9.2_linux_amd64/ks /usr/local/bin/


# Set USER env variable to workaround a known issue
# https://github.com/ksonnet/ksonnet/blob/master/docs/troubleshooting.md#error-user-current-not-implemented-on-linuxamd64
ENV USER=minikube

# Copy bootstrapper files
COPY ./minikube_bootstrapper.sh /
RUN chmod +x /minikube_bootstrapper.sh

ENTRYPOINT ["top", "-b"]