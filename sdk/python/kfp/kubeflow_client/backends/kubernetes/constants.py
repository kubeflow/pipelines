# Copyright The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Constants for the Kubernetes backend."""

IN_CLUSTER_DNS_NAME = 'http://ml-pipeline.{}.svc.cluster.local:8888'
KUBE_PROXY_PATH = 'api/v1/namespaces/{}/services/ml-pipeline:http/proxy/'
NAMESPACE_PATH = '/var/run/secrets/kubernetes.io/serviceaccount/namespace'
DEFAULT_NAMESPACE = 'kubeflow'

KFP_SA_TOKEN_PATH = '/var/run/secrets/kubeflow/pipelines/token'
K8S_SA_TOKEN_PATH = '/var/run/secrets/kubernetes.io/serviceaccount/token'
TOKEN_PATH_ENV = 'KF_PIPELINES_SA_TOKEN_PATH'
ENDPOINT_ENV = 'KF_PIPELINES_ENDPOINT'
