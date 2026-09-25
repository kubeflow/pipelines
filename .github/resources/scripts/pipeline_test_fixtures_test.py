#!/usr/bin/env python3
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
"""Verify URL-import fixtures against the HTTP service's projected files."""

import functools
import http.server
from pathlib import Path
import re
import tempfile
import threading
import unittest
import urllib.request

ROOT = Path(__file__).resolve().parents[3]
FIXTURES = ROOT / '.github/resources/manifests/pipeline-test-fixtures'


class QuietHandler(http.server.SimpleHTTPRequestHandler):

    def log_message(self, format, *args):
        pass


class PipelineTestFixturesTest(unittest.TestCase):

    def test_serves_every_integration_pipeline_url(self):
        sources = dict(
            re.findall(r'^\s*- ([^=\s]+)=(.+)$',
                       (FIXTURES / 'kustomization.yaml').read_text(),
                       re.MULTILINE))
        projections = re.findall(r'- key: ([^\n]+)\n\s+path: ([^\n]+)',
                                 (FIXTURES / 'deployment.yaml').read_text())
        self.assertTrue(sources)
        self.assertTrue(projections)

        paths = []
        for source in (ROOT / 'backend/test/v2/integration').glob('*.go'):
            text = source.read_text()
            urls = re.findall(
                r'GetRepoBranchURLRAW\([^,\n]+,\s*[^,\n]+,\s*"([^"]+)"\)', text)
            self.assertEqual(
                text.count('GetRepoBranchURLRAW('), len(urls),
                f'Unrecognized fixture URL in {source}')
            paths.extend(urls)
        self.assertTrue(paths)

        with tempfile.TemporaryDirectory() as directory:
            for key, path in projections:
                destination = Path(directory) / path
                destination.parent.mkdir(parents=True, exist_ok=True)
                destination.write_bytes((FIXTURES / sources[key]).read_bytes())
            handler = functools.partial(QuietHandler, directory=directory)
            server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), handler)
            thread = threading.Thread(target=server.serve_forever, daemon=True)
            thread.start()
            try:
                for path in paths:
                    with self.subTest(path=path):
                        url = f'http://127.0.0.1:{server.server_port}/{path}'
                        with urllib.request.urlopen(url, timeout=5) as response:
                            self.assertEqual(response.status, 200)
                            self.assertEqual(response.read(),
                                             (ROOT / path).read_bytes())
            finally:
                server.shutdown()
                server.server_close()
                thread.join(timeout=5)


if __name__ == '__main__':
    unittest.main()
