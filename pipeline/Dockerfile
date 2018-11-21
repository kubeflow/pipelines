# This is a dockerfile for an image with kubectl and ksonnet installed
# Available at gcr.io/ml-pipeline/bootstrapper

FROM google/cloud-sdk

RUN apt-get update -y
RUN apt-get install -y curl

# Install ksonnet
ARG KS_VERSION="0.11.0"
RUN curl -LO https://github.com/ksonnet/ksonnet/releases/download/v${KS_VERSION}/ks_${KS_VERSION}_linux_amd64.tar.gz
RUN tar -xzf ks_${KS_VERSION}_linux_amd64.tar.gz
RUN chmod +x ./ks_${KS_VERSION}_linux_amd64/ks
RUN mv ./ks_${KS_VERSION}_linux_amd64/ks /usr/local/bin/

# Set USER env variable to workaround a known issue
# https://github.com/ksonnet/ksonnet/blob/master/docs/troubleshooting.md#error-user-current-not-implemented-on-linuxamd64
ENV USER=default

# Copy bootstrapper files
COPY . /pipeline/
RUN chmod +x /pipeline/bootstrapper.sh
RUN chmod +x /pipeline/deploy.sh

# install uuidgen. Used for reporting usage
RUN apt-get install -y uuid-runtime

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

ENTRYPOINT ["/pipeline/deploy.sh"]
CMD []
