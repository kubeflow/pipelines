#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name('list-image-artifacts.sh')


def response(names=(), *, status=200, etag='"one"', more=False, invalid=False):
    headers = f'HTTP/2.0 {status}\r\n'
    if etag:
        headers += f'ETag: {etag}\r\n'
    if more:
        headers += 'Link: <https://api.github.com/next>; rel="next"\r\n'
    return {
        'output': headers + '\r\n' + '\n'.join(names),
        'code': 0 if status == 200 and not invalid else 1,
    }


class ListImageArtifactsTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.cache = self.root / 'cache'
        self.cache.mkdir()
        fake = self.root / 'gh'
        fake.write_text("""#!/usr/bin/env python3
import json, os, pathlib, sys
root = pathlib.Path(os.environ['FAKE_ROOT'])
responses = json.loads((root / 'responses').read_text())
response = responses.pop(0)
(root / 'responses').write_text(json.dumps(responses))
with (root / 'calls').open('a') as stream:
    stream.write(json.dumps(sys.argv[1:]) + '\\n')
sys.stdout.write(response['output'])
sys.exit(response['code'])
""")
        fake.chmod(0o755)
        self.env = dict(
            os.environ,
            FAKE_ROOT=str(self.root),
            GITHUB_REPOSITORY='owner/repo',
            GITHUB_RUN_ID='123',
            PATH=f'{self.root}{os.pathsep}{os.environ["PATH"]}',
        )

    def responses(self, *responses):
        (self.root / 'responses').write_text(json.dumps(responses))

    def poll(self, success=True):
        result = subprocess.run(
            [
                'bash',
                '-c',
                'source "$1"; list_image_artifacts "$2"',
                'test',
                str(SCRIPT),
                str(self.cache),
            ],
            env=self.env,
            text=True,
            capture_output=True,
        )
        self.assertEqual(result.returncode == 0, success, result.stderr)
        if not success:
            self.assertEqual(result.stdout, '')
        return result.stdout.splitlines()

    def calls(self):
        return [
            json.loads(line)
            for line in (self.root / 'calls').read_text().splitlines()
        ]

    def test_200_then_304_then_changed_inventory(self):
        self.responses(
            response(['first']),
            response(status=304),
            response(['first', 'second'], etag='"two"'),
        )
        self.assertEqual(self.poll(), ['first'])
        self.assertEqual(self.poll(), ['first'])
        self.assertEqual(self.poll(), ['first', 'second'])
        self.assertNotIn('-H', self.calls()[0])
        self.assertIn('If-None-Match: "one"', self.calls()[1])

    def test_revalidates_every_page_and_obeys_new_last_page(self):
        self.responses(
            response(['one'], more=True),
            response(['two'], etag='"two"'),
            response(status=304),
            response(['three'], etag='"three"'),
            response(['only'], etag='"only"'),
        )
        self.assertEqual(self.poll(), ['one', 'two'])
        self.assertEqual(self.poll(), ['one', 'three'])
        self.assertIn('page=2', self.calls()[3][2])
        self.assertIn('If-None-Match: "two"', self.calls()[3])
        self.assertEqual(self.poll(), ['only'])
        self.assertEqual(len(self.calls()), 5)

    def test_error_never_returns_cached_or_partial_success(self):
        self.responses(
            response(['old']),
            response(['new'], more=True),
            response(status=500),
        )
        self.poll()
        self.poll(success=False)

    def test_304_without_exact_url_cache_fails(self):
        self.responses(response(['old']), response(status=304))
        self.poll()
        self.env['GITHUB_RUN_ID'] = '456'
        self.poll(success=False)
        self.assertNotIn('-H', self.calls()[1])

    def test_no_validator_falls_back_to_fresh_reads(self):
        self.responses(
            response(['old'], etag=None), response(['new'], etag=None)
        )
        self.poll()
        self.assertEqual(self.poll(), ['new'])
        self.assertNotIn('-H', self.calls()[1])

    def test_failed_json_extraction_is_not_cached(self):
        self.responses(response(['partial'], invalid=True))
        self.poll(success=False)
        self.assertFalse((self.cache / '1.body').exists())

    def test_empty_inventory_revalidates(self):
        self.responses(response(), response(status=304))
        self.assertEqual(self.poll(), [])
        self.assertEqual(self.poll(), [])

    def test_empty_lf_only_inventory_revalidates(self):
        initial = response()
        initial['output'] = initial['output'].replace('\r\n', '\n')
        self.responses(initial, response(status=304))
        self.assertEqual(self.poll(), [])
        self.assertEqual(self.poll(), [])

    def test_missing_headers_fail(self):
        self.responses({'output': 'frontend', 'code': 0})
        self.poll(success=False)

    def test_failed_cache_write_invalidates_old_inventory(self):
        self.responses(
            response(['old-ready']),
            response([], etag='"new"'),
            response(status=304),
        )
        self.assertEqual(self.poll(), ['old-ready'])
        body = self.cache / '1.body'
        old_body = body.read_text()
        body.unlink()
        body.mkdir()
        self.poll(success=False)
        self.assertEqual((self.cache / '1.etag').read_text(), '')
        # Even if the old payload becomes readable again, a subsequent 304
        # cannot certify it: no validator survived the failed update.
        body.rmdir()
        body.write_text(old_body)
        self.poll(success=False)
        self.assertNotIn('-H', self.calls()[2])


if __name__ == '__main__':
    unittest.main()
