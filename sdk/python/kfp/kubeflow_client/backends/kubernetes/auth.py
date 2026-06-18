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
"""Authentication helpers for the Kubernetes backend."""

from __future__ import annotations

import logging
import os

from kfp.kubeflow_client.backends.kubernetes import constants
import kfp_server_api

logger = logging.getLogger(__name__)


def apply_in_cluster_credentials(
    api_config: kfp_server_api.Configuration,) -> None:
    """Apply default in-cluster service account credentials."""
    token_path = os.environ.get(constants.TOKEN_PATH_ENV)
    if not token_path:
        if os.path.exists(constants.KFP_SA_TOKEN_PATH):
            token_path = constants.KFP_SA_TOKEN_PATH
        elif os.path.exists(constants.K8S_SA_TOKEN_PATH):
            token_path = constants.K8S_SA_TOKEN_PATH
            logger.debug(
                'Kubeflow pipelines token path missing; using Kubernetes '
                'service account token at %s for API auth.', token_path)

    if not token_path:
        logger.debug(
            'No in-cluster token file found; skipping credential setup.')
        return

    try:
        from kfp.client.set_volume_credentials import ServiceAccountTokenVolumeCredentials
    except ImportError:
        logger.debug(
            'In-cluster credential module not available.', exc_info=True)
        return
    try:
        credentials = ServiceAccountTokenVolumeCredentials(path=token_path)
        credentials.refresh_api_key_hook(api_config)
        api_config.api_key_prefix['authorization'] = 'Bearer'
        api_config.refresh_api_key_hook = credentials.refresh_api_key_hook
    except FileNotFoundError:
        logger.warning(
            'Token file not found; proceeding without authentication.',
            exc_info=True)


def refresh_credentials(api_config: kfp_server_api.Configuration,) -> None:
    """Refresh the API token using the configured refresh hook."""
    if api_config.refresh_api_key_hook is not None:
        api_config.refresh_api_key_hook(api_config)
    else:
        raise RuntimeError('Token expired but no refresh hook is configured. '
                           'Re-create the client with a fresh token.')
