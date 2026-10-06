#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Authenticated loopback HTTP transport for the mutating CI fixture."""

import http.client
import ipaddress
import json
import re
import time
import urllib.parse

MAX_BYTES = 4 * 1024 * 1024


class FixtureError(ValueError):
    """A redacted fixture preparation error."""


class FixtureClient:
    """Authenticated POSTs to a literal loopback port-forward, without
    proxies."""

    def __init__(self, endpoint, token_file):
        try:
            parsed = urllib.parse.urlsplit(endpoint)
            if (parsed.scheme != 'http' or parsed.username is not None or
                    parsed.password is not None or
                    parsed.path not in ('', '/') or parsed.query or
                    parsed.fragment or not parsed.port or
                    not ipaddress.ip_address(parsed.hostname).is_loopback):
                raise ValueError()
            self.host, self.port = parsed.hostname, parsed.port
            with open(token_file, 'rb') as stream:
                raw = stream.read(16385)
            token = raw.decode('ascii').strip()
            if len(raw) > 16384 or not re.fullmatch(r'[A-Za-z0-9._~+/-]+=*',
                                                    token):
                raise ValueError()
            self.token = token
        except (OSError, ValueError, TypeError):
            raise FixtureError('invalid_loopback_endpoint_or_token') from None

    def post(self, path, body):
        if not re.fullmatch(
                r'/apis/v2beta1/(experiments|recurringruns)(/[a-zA-Z0-9_.-]+:(enable|disable))?',
                path):
            raise FixtureError('invalid_fixture_api_path')
        payload = json.dumps(body).encode('utf-8')
        if len(payload) > MAX_BYTES:
            raise FixtureError('request_limit_exceeded')
        connection = http.client.HTTPConnection(
            self.host, self.port, timeout=20)
        try:
            deadline = time.monotonic() + 20
            connection.request(
                'POST',
                path,
                body=payload,
                headers={
                    'Authorization': 'Bearer ' + self.token,
                    'Content-Type': 'application/json',
                    'Accept-Encoding': 'identity'
                })
            response = connection.getresponse()
            if response.status < 200 or response.status >= 300:
                raise FixtureError('fixture_api_request_rejected')
            chunks, size = [], 0
            while True:
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    raise FixtureError('fixture_api_timeout')
                raw = getattr(getattr(response, 'fp', None), 'raw', None)
                sock = getattr(raw, '_sock', None) or connection.sock
                if sock is not None:
                    sock.settimeout(remaining)
                chunk = response.read1(min(65536, MAX_BYTES - size + 1))
                size += len(chunk)
                if size > MAX_BYTES or time.monotonic() >= deadline:
                    raise FixtureError('fixture_api_response_limit')
                if not chunk:
                    break
                chunks.append(chunk)
            value = json.loads(b''.join(chunks) or b'{}')
            if not isinstance(value, dict) or value.get('error'):
                raise FixtureError('invalid_fixture_api_response')
            return value
        except (OSError, http.client.HTTPException, ValueError):
            raise FixtureError('fixture_api_request_failed') from None
        finally:
            connection.close()
