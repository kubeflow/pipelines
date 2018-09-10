# This image is used to run ML pipeline CI test.
# Available at gcr.io/ml-pipeline/test-worker

FROM gcr.io/k8s-testimages/kubekins-e2e:latest-master

# Install Argo
RUN curl -sSL -o /usr/local/bin/argo https://github.com/argoproj/argo/releases/download/v2.2.0/argo-linux-amd64
RUN chmod +x /usr/local/bin/argo
