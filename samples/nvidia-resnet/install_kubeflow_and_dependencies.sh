#!/bin/bash
# Copyright (c) 2019, NVIDIA CORPORATION. All rights reserved.
#
# Redistribution and use in source and binary forms, with or without
# modification, are permitted provided that the following conditions
# are met:
#  * Redistributions of source code must retain the above copyright
#    notice, this list of conditions and the following disclaimer.
#  * Redistributions in binary form must reproduce the above copyright
#    notice, this list of conditions and the following disclaimer in the
#    documentation and/or other materials provided with the distribution.
#  * Neither the name of NVIDIA CORPORATION nor the names of its
#    contributors may be used to endorse or promote products derived
#    from this software without specific prior written permission.
#
# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS ``AS IS'' AND ANY
# EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
# IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
# PURPOSE ARE DISCLAIMED.  IN NO EVENT SHALL THE COPYRIGHT OWNER OR
# CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
# EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
# PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
# PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY
# OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
# (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
# OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.


# Install nvidia-docker 2
sudo tee /etc/docker/daemon.json <<EOF
{
    "default-runtime": "nvidia",
    "runtimes": {
        "nvidia": {
            "path": "/usr/bin/nvidia-container-runtime",
            "runtimeArgs": []
        }
    }
}
EOF
sudo pkill -SIGHUP dockerd

# Install MiniKube
curl -Lo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 \
  && chmod +x minikube
sudo cp minikube /usr/local/bin && rm minikube

# Install kubectl
sudo apt-get update && sudo apt-get install -y apt-transport-https
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
sudo touch /etc/apt/sources.list.d/kubernetes.list
echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee -a /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update
sudo apt-get install -y kubectl

# Install ksonnet
export KS_VER=0.13.1
wget https://github.com/ksonnet/ksonnet/releases/download/v${KS_VER}/ks_${KS_VER}_linux_amd64.tar.gz -O ksonnet.tar.gz
mkdir -p ksonnet
tar -xvf ksonnet.tar.gz -C ksonnet --strip-components=1
sudo cp ksonnet/ks /usr/local/bin
rm -fr ksonnet
rm ksonnet.tar.gz

# Start Minikube
sudo minikube start --vm-driver=none --extra-config=kubelet.resolv-conf=/run/systemd/resolve/resolv.conf

# Add the NVIDIA device plugin
kubectl create -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v1.11/nvidia-device-plugin.yml

# Install Kubeflow
export KUBEFLOW_SRC=/tmp/kubeflow
export KFAPP=kfapp
mkdir ${KUBEFLOW_SRC}
cd ${KUBEFLOW_SRC}
export KUBEFLOW_TAG=v0.4.1
curl https://raw.githubusercontent.com/kubeflow/kubeflow/${KUBEFLOW_TAG}/scripts/download.sh | bash
KUBEFLOW_REPO=${KUBEFLOW_SRC} ${KUBEFLOW_SRC}/scripts/kfctl.sh init ${KFAPP} --platform minikube
cd ${KFAPP}
${KUBEFLOW_SRC}/scripts/kfctl.sh generate all
${KUBEFLOW_SRC}/scripts/kfctl.sh apply all

# Open up ambassador’s port
kubectl patch svc ambassador -n kubeflow --type='json' -p '[{"op":"replace","path":"/spec/type","value":"NodePort"}]'
