"""The `kfp.client` module contains the KFP API client."""
# Copyright 2022 The Kubeflow Authors
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

__all__ = [
    'Client',
]

from kfp.client.client import Client
from kfp.client.set_volume_credentials import \
    ServiceAccountTokenVolumeCredentials
from kfp.client.token_credentials_base import TokenCredentialsBase

KF_PIPELINES_SA_TOKEN_ENV = 'KF_PIPELINES_SA_TOKEN_PATH'
KF_PIPELINES_SA_TOKEN_PATH = '/var/run/secrets/kubeflow/pipelines/token'
