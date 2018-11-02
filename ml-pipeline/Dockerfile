# This is a dockerfile for an image with kubectl and ksonnet installed
# Available at gcr.io/ml-pipeline/bootstrapper

FROM ubuntu:18.04

RUN apt-get update -y
RUN apt-get install -y curl

# Install kubectl
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv kubectl /usr/local/bin/

# Install ksonnet
RUN curl -LO https://github.com/ksonnet/ksonnet/releases/download/v0.11.0/ks_0.11.0_linux_amd64.tar.gz
RUN tar -xzf ks_0.11.0_linux_amd64.tar.gz
RUN chmod +x ./ks_0.11.0_linux_amd64/ks
RUN mv ./ks_0.11.0_linux_amd64/ks /usr/local/bin/


# Set USER env variable to workaround a known issue
# https://github.com/ksonnet/ksonnet/blob/master/docs/troubleshooting.md#error-user-current-not-implemented-on-linuxamd64
ENV USER=default

# Copy bootstrapper files
COPY . /ml-pipeline/
RUN chmod +x /ml-pipeline/bootstrapper.sh
RUN chmod +x /ml-pipeline/deploy.sh

# install uuidgen. Used for reporting usage
RUN apt-get install uuid-runtime

# Specify the release version and images as a build time arguments so that it can be specified in the `docker build ...` command.
ARG RELEASE_VERSION
ARG API_SERVER_IMAGE
ARG SCHEDULED_WORKFLOW_IMAGE
ARG PERSISTENCE_AGENT_IMAGE
ARG UI_IMAGE

# Additionally specify the release version and images as environment variables so that they are available to the
# `deploy.sh` script at runtime. The default values of each environment variable is the value of the build arg.
ENV RELEASE_VERSION ${RELEASE_VERSION}
ENV API_SERVER_IMAGE ${API_SERVER_IMAGE}
ENV SCHEDULED_WORKFLOW_IMAGE ${SCHEDULED_WORKFLOW_IMAGE}
ENV PERSISTENCE_AGENT_IMAGE ${PERSISTENCE_AGENT_IMAGE}
ENV UI_IMAGE ${UI_IMAGE}

ENTRYPOINT ["/ml-pipeline/deploy.sh"]
CMD []
