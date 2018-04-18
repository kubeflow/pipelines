#!/bin/bash

# When kubectl is installed in a k8s pod, it uses env variables instead of config file to talk with master node
# https://kubernetes.io/docs/concepts/services-networking/connect-applications-service/#accessing-the-service.
# The kubeconfig file is empty.
# Generate kubeconfig file based on the env variable for ksonnet to connect to K8s API inside a pod.
mkdir /root/.kube

export KUBERNETES_TOKEN=`cat /var/run/secrets/kubernetes.io/serviceaccount/token`

cat >/root/.kube/config <<EOF
apiVersion: v1
clusters:
- cluster:
    certificate-authority: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    server: https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}
  name: default
contexts:
- context:
    cluster: default
    user: default
  name: default
current-context: default
kind: Config
preferences: {}
users:
- name: default
  user:
    token: ${KUBERNETES_TOKEN}
EOF

