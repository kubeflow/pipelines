#!/bin/bash

# This file is a bootstrapper to configure the container when it's running in minikube,
# to allow ksonnet run in the container

mkdir /root/.kube

export KUBERNETES_TOKEN=`cat /var/run/secrets/kubernetes.io/serviceaccount/token`

# When kubectl is installed in a k8s pod, it uses env variables instead of config file to talk with master node
# https://stackoverflow.com/a/42651673
# However ksonnet requires a kubeconfig file currently.
# Generate kubeconfig file based on the env variable
cat >/root/.kube/config <<EOF
apiVersion: v1
clusters:
- cluster:
    certificate-authority: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    server: https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}
  name: minikube
contexts:
- context:
    cluster: minikube
    user: minikube
  name: minikube
current-context: minikube
kind: Config
preferences: {}
users:
- name: minikube
  user:
    token: ${KUBERNETES_TOKEN}
EOF

