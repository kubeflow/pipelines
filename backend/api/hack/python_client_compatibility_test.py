# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Check a standalone installed client against its generated requirements."""

from http.server import BaseHTTPRequestHandler
from http.server import HTTPServer
import importlib.metadata
import json
import os
from pathlib import Path
import threading
import unittest

from kfp_server_api import Configuration
from kfp_server_api.rest import ApiException
from kfp_server_api.rest import RESTClientObject
from packaging.requirements import Requirement
from packaging.specifiers import SpecifierSet


class PythonClientCompatibilityTest(unittest.TestCase):

    def test_installed_wheel_metadata(self):
        metadata = importlib.metadata.metadata('kfp-server-api')
        supported = SpecifierSet(metadata['Requires-Python'] or '')
        self.assertNotIn('3.8', supported)
        self.assertIn('3.9', supported)
        self.assertIn('3.10', supported)
        requirements = [
            Requirement(r) for r in metadata.get_all('Requires-Dist')
        ]
        self.assertNotIn('setuptools', [r.name for r in requirements])
        self.check_floors(requirements, 'urllib3', '2.6.3', '2.8.0')

    def test_generated_requirement_floors(self):
        root = Path(os.environ['CLIENT_ROOT'])
        requirements = [
            Requirement(line)
            for line in (root / 'requirements.txt').read_text().splitlines()
            if line.strip() and not line.startswith('#')
        ]
        self.check_floors(requirements, 'urllib3', '2.6.3', '2.8.0')
        self.check_floors(requirements, 'setuptools', '82.0.1', '84.1.0')
        self.assertNotIn('future', [r.name for r in requirements])

    def check_floors(self, requirements, package, old_floor, new_floor):
        for python, floor in [('3.9', old_floor), ('3.10', new_floor)]:
            with self.subTest(package=package, python=python):
                active = [
                    r for r in requirements
                    if r.name == package and (not r.marker or r.marker.evaluate(
                        {'python_version': python}))
                ]
                self.assertEqual(len(active), 1)
                self.assertIn(floor, active[0].specifier)
                previous = {
                    '2.6.3': '2.6.2',
                    '2.8.0': '2.7.0',
                    '82.0.1': '82.0.0',
                    '84.1.0': '84.0.0',
                }[floor]
                self.assertNotIn(previous, active[0].specifier)
                if python == '3.10':
                    self.assertNotIn(old_floor, active[0].specifier)

    def test_http_transport(self):

        class Handler(BaseHTTPRequestHandler):

            def do_GET(self):
                payload = json.dumps({'path': self.path}).encode()
                self.send_response(404 if self.path == '/missing' else 200)
                self.send_header('Content-Type', 'application/json')
                self.send_header('Content-Length', str(len(payload)))
                self.end_headers()
                self.wfile.write(payload)

            def log_message(self, *args):
                pass

        with HTTPServer(('127.0.0.1', 0), Handler) as server:
            worker = threading.Thread(target=server.serve_forever, daemon=True)
            worker.start()
            client = RESTClientObject(Configuration())
            base = f'http://127.0.0.1:{server.server_port}'
            try:
                response = client.request(
                    'GET',
                    base + '/runs',
                    query_params=[('name', 'a b')],
                    _request_timeout=5)
                self.assertEqual(response.status, 200)
                self.assertEqual(
                    json.loads(response.data), {'path': '/runs?name=a+b'})
                self.assertEqual(
                    response.getheader('Content-Type'), 'application/json')
                with self.assertRaises(ApiException) as caught:
                    client.request('GET', base + '/missing', _request_timeout=5)
                self.assertEqual(caught.exception.status, 404)
                self.assertEqual(
                    json.loads(caught.exception.body), {'path': '/missing'})
            finally:
                client.pool_manager.clear()
                server.shutdown()
                worker.join(timeout=5)


if __name__ == '__main__':
    unittest.main()
