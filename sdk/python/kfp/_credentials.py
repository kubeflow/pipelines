# Copyright 2021 Arrikto Inc.
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

import os
import abc

from kubernetes.client.configuration import Configuration


__all__ = [
    "TokenCredentialsBase",
    "ML_PIPELINE_SA_TOKEN_ENV",
    "ML_PIPELINE_SA_TOKEN_PATH",
    "ServiceAccountTokenVolumeCredentials",
]


ML_PIPELINE_SA_TOKEN_ENV = "ML_PIPELINE_SA_TOKEN_PATH"
ML_PIPELINE_SA_TOKEN_PATH = "/var/run/secrets/ml-pipeline/token"


class TokenCredentialsBase(abc.ABC):

    def refresh_api_key_hook(self, config: Configuration):
        """Refresh the api key.

        This is a helper function for registering token refresh with swagger
        generated clients.

        Args:
            config (kubernetes.client.configuration.Configuration):
                The configuration object that the client uses.

                The Configuration object of the kubernetes client's is the same
                with kfp_server_api.configuration.Configuration.
        """
        config.api_key["authorization"] = self.get_token()

    @abc.abstractmethod
    def get_token(self):
        raise NotImplementedError()


def read_sa_token(path=None):
    """Read a ServiceAccount token found under some path."""
    token = None
    with open(path, "r") as f:
        token = f.read().strip()
    return token


class ServiceAccountTokenVolumeCredentials(TokenCredentialsBase):
    """Audience-bound ServiceAccountToken in the local filesystem.

    This is a credentials interface for audience-bound ServiceAccountTokens
    found in the local filesystem, that get refreshed by the kubelet.

    The constructor of the class expects a filesystem path.
    If not provided, it uses the path stored in the environment variable
    defined in ML_PIPELINE_SA_TOKEN_ENV.
    If the environment variable is also empty, it falls back to the path
    specified in ML_PIPELINE_SA_TOKEN_PATH.

    This method of authentication is meant for use inside a Kubernetes cluster.

    Relevant documentation:
    https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection
    """

    def __init__(self, path=None):
        self.token_path = (path
                           or os.getenv(ML_PIPELINE_SA_TOKEN_ENV)
                           or ML_PIPELINE_SA_TOKEN_PATH)

    def get_token(self):
        return read_sa_token(self.token_path)
