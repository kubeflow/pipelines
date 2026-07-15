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
"""Type definitions for the Kubernetes backend."""

from __future__ import annotations

import dataclasses


@dataclasses.dataclass
class KubernetesBackendConfig:
    """Connection configuration for the KFP API server.

    Args:
        base_url: KFP API server URL including scheme and port
            (e.g. ``https://ml-pipeline.example.com:8080``). If omitted,
            auto-discovered following kfp.Client conventions (in-cluster DNS
            or kubeconfig proxy).
        user_token: Bearer token for authentication.
        is_secure: Whether to verify TLS certificates (controls
            ``verify_ssl`` on the underlying HTTP client). Does not control
            whether the connection uses TLS — that is determined by the URL
            scheme. Inferred from scheme if omitted (``True`` for https,
            ``False`` for http).
        custom_ca: Path to PEM-encoded root certificates.
        namespace: Kubernetes namespace. If omitted, auto-detected.
    """

    base_url: str | None = None
    user_token: str | None = None
    is_secure: bool | None = None
    custom_ca: str | None = None
    namespace: str | None = None

    def __repr__(self) -> str:
        token_display = '***' if self.user_token else None
        return (f'KubernetesBackendConfig('
                f'base_url={self.base_url!r}, '
                f'user_token={token_display!r}, '
                f'is_secure={self.is_secure!r}, '
                f'custom_ca={self.custom_ca!r}, '
                f'namespace={self.namespace!r})')
