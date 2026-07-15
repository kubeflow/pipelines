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
"""Utility helpers for the Kubernetes backend."""

from __future__ import annotations

import logging
import os
import ssl

from kfp.kubeflow_client.backends.kubernetes import constants

logger = logging.getLogger(__name__)

_COMMON_CA_BUNDLE_PATHS = (
    '/etc/ssl/certs/ca-certificates.crt',
    '/etc/pki/tls/certs/ca-bundle.crt',
    '/etc/ssl/ca-bundle.pem',
    '/etc/ssl/cert.pem',
)


def detect_system_ca_bundle() -> str | None:
    """Best-effort detection of a system CA certificate bundle.

    Resolution order:
        1. ``SSL_CERT_FILE`` environment variable.
        2. ``REQUESTS_CA_BUNDLE`` environment variable.
        3. OpenSSL default CA file (via :func:`ssl.get_default_verify_paths`).
        4. Common OS bundle paths (Debian, RHEL, openSUSE, macOS/Alpine).

    Returns:
        Absolute path to a CA bundle file, or ``None`` if no bundle was found.
    """
    for env_var in ('SSL_CERT_FILE', 'REQUESTS_CA_BUNDLE'):
        path = os.environ.get(env_var)
        if path and os.path.isfile(path):
            logger.debug('System CA bundle from %s: %s', env_var, path)
            return path

    try:
        defaults = ssl.get_default_verify_paths()
        if defaults.cafile and os.path.isfile(defaults.cafile):
            logger.debug('System CA bundle from OpenSSL defaults: %s',
                         defaults.cafile)
            return defaults.cafile
    except Exception:  # pylint: disable=broad-except
        logger.debug('ssl.get_default_verify_paths() failed.', exc_info=True)

    for path in _COMMON_CA_BUNDLE_PATHS:
        if os.path.isfile(path):
            logger.debug('System CA bundle found at: %s', path)
            return path

    return None


def discover_host(namespace: str) -> str:
    """Auto-discover the KFP API server endpoint."""
    endpoint_from_env = os.environ.get(constants.ENDPOINT_ENV)
    if endpoint_from_env:
        host = endpoint_from_env.rstrip('/')
        if not (host.startswith('http://') or host.startswith('https://')):
            logger.warning(
                'No scheme in KF_PIPELINES_ENDPOINT %r, defaulting '
                'to https.', endpoint_from_env)
            host = 'https://' + host
        return host

    try:
        import kubernetes as k8s
    except ImportError:
        if os.path.exists(constants.NAMESPACE_PATH):
            return constants.IN_CLUSTER_DNS_NAME.format(namespace)
        raise ValueError(
            'Could not auto-discover KFP endpoint: the kubernetes package '
            'is not installed and no in-cluster environment was detected. '
            'Set base_url in KubernetesBackendConfig or the '
            'KF_PIPELINES_ENDPOINT environment variable.')

    try:
        k8s.config.load_incluster_config()
        return constants.IN_CLUSTER_DNS_NAME.format(namespace)
    except (k8s.config.ConfigException, FileNotFoundError):
        logger.debug('In-cluster config not available.', exc_info=True)

    # Only the host URL is extracted from kubeconfig; kubeconfig auth
    # credentials are not applied to the API configuration. This matches
    # kfp.Client behavior and assumes kubectl proxy handles auth.
    try:
        k8s_config = k8s.client.Configuration()
        k8s.config.load_kube_config(client_configuration=k8s_config)
        if k8s_config.host:
            return (k8s_config.host.rstrip('/') + '/' +
                    constants.KUBE_PROXY_PATH.format(namespace))
    except (k8s.config.ConfigException, FileNotFoundError):
        logger.debug('Kubeconfig not available.', exc_info=True)

    fallback = constants.IN_CLUSTER_DNS_NAME.format(namespace)
    logger.warning(
        'Could not detect KFP endpoint via in-cluster config or '
        'kubeconfig. Falling back to %s. Set base_url in '
        'KubernetesBackendConfig or KF_PIPELINES_ENDPOINT to override.',
        fallback)
    return fallback


def resolve_namespace(configured_namespace: str | None) -> str:
    """Return the configured namespace, auto-detecting if needed.

    Resolution order:
        1. Explicitly configured via ``KubernetesBackendConfig.namespace``.
        2. In-cluster: ``/var/run/secrets/kubernetes.io/serviceaccount/namespace``.
        3. Out-of-cluster: namespace from the current kubeconfig context.
        4. Fallback: ``"kubeflow"``.

    Note: Does not read ~/.config/kfp/context.json (used by kfp.Client's
    set_user_namespace). This will be implemented in further phases.
    """
    if configured_namespace:
        return configured_namespace

    try:
        with open(constants.NAMESPACE_PATH, 'r') as f:
            return f.read().strip()
    except FileNotFoundError:
        pass

    try:
        import kubernetes as k8s
    except ImportError:
        logger.debug('kubernetes package not installed.')
    else:
        try:
            _, active_context = k8s.config.list_kube_config_contexts()
            namespace = active_context.get('context', {}).get('namespace')
            if namespace:
                logger.debug('Namespace resolved from kubeconfig context: %r.',
                             namespace)
                return namespace
        except (k8s.config.ConfigException, FileNotFoundError):
            logger.debug(
                'Could not read namespace from kubeconfig.', exc_info=True)

    logger.debug(
        'Namespace not resolved from cluster or kubeconfig; '
        'using default %r.', constants.DEFAULT_NAMESPACE)
    return constants.DEFAULT_NAMESPACE
