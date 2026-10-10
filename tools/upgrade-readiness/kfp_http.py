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
"""Bounded, authenticated, read-only transport for KFP readiness collection."""

from functools import partial
import http.client
import io
import ipaddress
import json
import re
import ssl
import time
import urllib.error
import urllib.parse
import urllib.request

TIMEOUT_SECONDS = 20
MAX_RESPONSE_BYTES = 16 * 1024 * 1024
MAX_TOTAL_BYTES = 16 * 1024 * 1024
MAX_REQUESTS = 200
MAX_TOKEN_BYTES = 16 * 1024


class CollectionError(ValueError):
    """A sanitized collection failure; reasons never contain server data."""

    def __init__(self, reason):
        self.reason = reason
        super().__init__(reason)


class _NoRedirect(urllib.request.HTTPRedirectHandler):

    def redirect_request(self, req, fp, code, msg, headers, newurl):
        fp.close()
        raise CollectionError('redirect_refused')


class _DeadlineReader(io.RawIOBase):
    """Apply the remaining budget to every receive, including HTTP framing."""

    def __init__(self, raw, sock, deadline):
        super().__init__()
        self._raw = raw
        self._socket = sock
        self._deadline = deadline

    def readable(self):
        return True

    def readinto(self, buffer):
        remaining = self._deadline - time.monotonic()
        if remaining <= 0:
            raise TimeoutError()
        self._socket.settimeout(remaining)
        count = self._raw.readinto(buffer)
        if time.monotonic() >= self._deadline:
            raise TimeoutError()
        return count

    def close(self):
        try:
            self._raw.close()
        finally:
            super().close()


class _DeadlineResponse(http.client.HTTPResponse):

    def __init__(self, sock, *args, deadline, **kwargs):
        super().__init__(sock, *args, **kwargs)
        # No bytes have been read yet. Retain the makefile stream's socket
        # ownership when urllib closes its connection after reading headers.
        self.fp = io.BufferedReader(
            _DeadlineReader(self.fp.detach(), sock, deadline))


def _deadline_connection(connection_type, deadline, *args, **kwargs):
    connection = connection_type(*args, **kwargs)
    connection.response_class = partial(_DeadlineResponse, deadline=deadline)
    return connection


class _DeadlineHTTPHandler(urllib.request.HTTPHandler):

    def http_open(self, request):
        return self.do_open(
            partial(_deadline_connection, http.client.HTTPConnection,
                    request.readiness_deadline), request)


class _DeadlineHTTPSHandler(urllib.request.HTTPSHandler):

    def https_open(self, request):
        return self.do_open(
            partial(_deadline_connection, http.client.HTTPSConnection,
                    request.readiness_deadline),
            request,
            context=self._context)


def _safe_path(path):
    decoded = urllib.parse.unquote(path)
    return (not any(ord(c) < 33 or ord(c) > 126 for c in path) and
            not any(c in decoded for c in ('\\', '?', '#')) and
            not any(ord(c) < 32 or ord(c) == 127 for c in decoded) and
            not any(part in ('.', '..') for part in decoded.split('/')))


class Client:
    """GET JSON from one endpoint without redirects or environment proxies."""

    def __init__(self, endpoint, token_file=None, ca_file=None):
        try:
            if (not isinstance(endpoint, str) or not endpoint or
                    any(ord(c) < 33 or ord(c) > 126 for c in endpoint)):
                raise ValueError()
            parsed = urllib.parse.urlsplit(endpoint)
            if (parsed.scheme not in ('https', 'http') or not parsed.hostname or
                    parsed.username is not None or
                    parsed.password is not None or '?' in endpoint or
                    '#' in endpoint or not _safe_path(parsed.path)):
                raise ValueError()
            # Force validation of malformed or out-of-range ports.
            if parsed.port is not None and parsed.port == 0:
                raise ValueError()
            if parsed.scheme == 'http':
                if not ipaddress.ip_address(parsed.hostname).is_loopback:
                    raise ValueError()
            self.endpoint = endpoint.rstrip('/')
        except (ValueError, TypeError):
            raise CollectionError('invalid_endpoint') from None
        self._token = None
        if token_file is not None:
            try:
                with open(token_file, 'rb') as stream:
                    raw = stream.read(MAX_TOKEN_BYTES + 1)
                token = raw.decode('ascii').strip()
                if len(raw) > MAX_TOKEN_BYTES or not re.fullmatch(
                        r'[A-Za-z0-9._~+/-]+=*', token):
                    raise ValueError()
                self._token = token
            except (OSError, ValueError):
                raise CollectionError('invalid_token_file') from None
        try:
            context = ssl.create_default_context(cafile=ca_file)
            self._opener = urllib.request.build_opener(
                urllib.request.ProxyHandler({}), _NoRedirect(),
                _DeadlineHTTPHandler(), _DeadlineHTTPSHandler(context=context))
        except (OSError, ValueError, ssl.SSLError):
            raise CollectionError('invalid_ca_file') from None
        self._requests = 0
        self._bytes = 0

    def get(self, path, params=None):
        """Read one API object, returning only sanitized errors on failure."""
        if (not isinstance(path, str) or
                not (path.startswith('/apis/v2beta1/') or re.fullmatch(
                    r'/apis/v1beta1/runs/[A-Za-z0-9._~-]+', path)) or
                not _safe_path(path) or '?' in path or '#' in path):
            raise CollectionError('invalid_api_path')
        if self._requests >= MAX_REQUESTS:
            raise CollectionError('request_budget_exceeded')
        if self._bytes >= MAX_TOTAL_BYTES:
            raise CollectionError('total_response_limit_exceeded')
        try:
            query = urllib.parse.urlencode(params or {})
        except (TypeError, ValueError):
            raise CollectionError('invalid_query') from None
        url = self.endpoint + path + ('?' + query if query else '')
        headers = {'Accept': 'application/json', 'Accept-Encoding': 'identity'}
        if self._token is not None:
            headers['Authorization'] = 'Bearer ' + self._token
        request = urllib.request.Request(url, headers=headers, method='GET')
        self._requests += 1
        deadline = time.monotonic() + TIMEOUT_SECONDS
        request.readiness_deadline = deadline
        chunks = []
        size = 0
        try:
            with self._opener.open(
                    request, timeout=TIMEOUT_SECONDS) as response:
                if response.status != 200:
                    raise CollectionError('unexpected_http_status')
                read = getattr(response, 'read1', response.read)
                while True:
                    remaining_time = deadline - time.monotonic()
                    if remaining_time <= 0:
                        raise CollectionError('request_timeout')
                    remaining = min(MAX_RESPONSE_BYTES - size,
                                    MAX_TOTAL_BYTES - self._bytes)
                    chunk = read(min(65536, remaining + 1))
                    if time.monotonic() >= deadline:
                        raise CollectionError('request_timeout')
                    size += len(chunk)
                    self._bytes += len(chunk)
                    if size > MAX_RESPONSE_BYTES:
                        raise CollectionError('response_limit_exceeded')
                    if self._bytes > MAX_TOTAL_BYTES:
                        raise CollectionError('total_response_limit_exceeded')
                    if not chunk:
                        break
                    chunks.append(chunk)
            value = json.loads(b''.join(chunks))
            if not isinstance(value, dict):
                raise CollectionError('invalid_json_object')
            if value.get('error'):
                raise CollectionError('api_error')
            return value
        except CollectionError:
            raise
        except urllib.error.HTTPError as error:
            error.close()
            if 300 <= error.code < 400:
                reason = 'redirect_refused'
            elif error.code in (401, 403):
                reason = 'access_denied'
            elif error.code == 404:
                reason = 'not_found'
            else:
                reason = 'http_error'
            raise CollectionError(reason) from None
        except TimeoutError:
            raise CollectionError('request_timeout') from None
        except (UnicodeError, ValueError, RecursionError):
            raise CollectionError('invalid_json_object') from None
        except (OSError, urllib.error.URLError, http.client.HTTPException):
            raise CollectionError('request_failed') from None
