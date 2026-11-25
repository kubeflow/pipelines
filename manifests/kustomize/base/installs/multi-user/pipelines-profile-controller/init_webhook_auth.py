# Copyright 2026 The Kubeflow Authors
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
"""Create the profile webhook credential once and preserve it across
upgrades."""

import base64
import json
import os
from pathlib import Path
import secrets
import ssl
import subprocess
import tempfile
import time
from urllib.error import HTTPError
from urllib.parse import quote
from urllib.request import build_opener
from urllib.request import HTTPSHandler
from urllib.request import ProxyHandler
from urllib.request import Request

SECRET_NAME = 'kubeflow-pipelines-profile-webhook-auth'
SECRET_KEY = 'token'
SERVICE_ACCOUNT = Path('/var/run/secrets/kubernetes.io/serviceaccount')
RETRYABLE_STATUS = {404, 409, 429, 500, 502, 503, 504}

TLS_DIR = Path('/var/run/kfp-webhook-tls')
TLS_KEYS = {'tls.crt', 'tls.key', 'ca.crt'}


def generate_tls_data(namespace, directory):
    """Pin a per-install certificate for the Service and local port forwards."""
    dns = f'kubeflow-pipelines-profile-controller.{namespace}'
    with tempfile.TemporaryDirectory(dir=directory) as temporary:
        key = Path(temporary) / 'tls.key'
        cert = Path(temporary) / 'tls.crt'
        subprocess.run(
            [
                'openssl',
                'req',
                '-x509',
                '-newkey',
                'rsa:3072',
                '-nodes',
                '-days',
                '365',
                '-subj',
                '/CN=KFP profile webhook',
                '-addext',
                f'subjectAltName=DNS:{dns},DNS:{dns}.svc,IP:127.0.0.1',
                '-addext',
                'basicConstraints=critical,CA:FALSE',
                '-addext',
                'keyUsage=critical,digitalSignature,keyEncipherment',
                '-addext',
                'extendedKeyUsage=serverAuth',
                '-keyout',
                str(key),
                '-out',
                str(cert),
            ],
            check=True,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        encoded_cert = base64.b64encode(cert.read_bytes()).decode('ascii')
        return {
            'tls.crt': encoded_cert,
            'tls.key': base64.b64encode(key.read_bytes()).decode('ascii'),
            'ca.crt': encoded_cert,
        }


def initialize_webhook_secret(request, namespace, directory, wait=time.sleep):
    """Fill a pre-created Secret using a resource-version guarded update."""
    for attempt in range(60):
        try:
            secret = request('GET', None)
            data = dict(secret.get('data') or {})
            existing = data.get(SECRET_KEY)
            if existing is not None:
                token = base64.b64decode(
                    existing, validate=True).decode('ascii')
                if (len(token) < 32 or not token.isascii() or
                        any(character.isspace() for character in token)):
                    raise ValueError('invalid existing webhook token')
            else:
                data[SECRET_KEY] = base64.b64encode(
                    secrets.token_urlsafe(32).encode('ascii')).decode('ascii')
            existing_tls = TLS_KEYS.intersection(data)
            if existing_tls and existing_tls != TLS_KEYS:
                raise ValueError('incomplete existing webhook TLS credentials')
            if not existing_tls:
                data.update(generate_tls_data(namespace, directory))
            if data == secret.get('data'):
                return data
            if not secret.get('metadata', {}).get('resourceVersion'):
                raise ValueError('missing Secret resource version')
            secret['data'] = data
            request('PUT', secret)
            return data
        except HTTPError as error:
            if error.code in RETRYABLE_STATUS and attempt < 59:
                wait(1)
                continue
            raise RuntimeError(
                'profile webhook Secret initialization failed') from None


def main():
    namespace = (SERVICE_ACCOUNT / 'namespace').read_text().strip()
    service_token = (SERVICE_ACCOUNT / 'token').read_text().strip()
    host = os.environ['KUBERNETES_SERVICE_HOST']
    port = os.environ.get('KUBERNETES_SERVICE_PORT_HTTPS', '443')
    if ':' in host and not host.startswith('['):
        host = f'[{host}]'
    url = (f'https://{host}:{port}/api/v1/namespaces/'
           f'{quote(namespace, safe="")}/secrets/{SECRET_NAME}')
    context = ssl.create_default_context(cafile=str(SERVICE_ACCOUNT / 'ca.crt'))
    opener = build_opener(ProxyHandler({}), HTTPSHandler(context=context))

    def request(method, secret):
        body = None if secret is None else json.dumps(secret).encode('utf-8')
        headers = {'Authorization': f'Bearer {service_token}'}
        if body is not None:
            headers['Content-Type'] = 'application/json'
        request = Request(url, data=body, headers=headers, method=method)
        with opener.open(request, timeout=5) as response:
            return json.load(response)

    os.umask(0o077)
    data = initialize_webhook_secret(request, namespace, TLS_DIR)
    for name in ('tls.crt', 'tls.key'):
        (TLS_DIR / name).write_bytes(
            base64.b64decode(data[name], validate=True))


if __name__ == '__main__':
    try:
        main()
    except Exception:
        # API error bodies and Secret values must never reach pod logs.
        raise SystemExit(
            'Unable to initialize profile webhook Secret; check the Secret and scoped RBAC.'
        ) from None
