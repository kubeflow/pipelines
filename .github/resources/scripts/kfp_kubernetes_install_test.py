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
"""Exercise make's local SDK selection with conflicting same-version wheels."""

import json
import os
from pathlib import Path
import shlex
import subprocess
import tempfile
import unittest
import venv
import zipfile

ROOT = Path(__file__).resolve().parents[3]


def wheel(directory, name, version, dependencies=()):
    directory.mkdir(parents=True, exist_ok=True)
    stem = f'{name}-{version}'
    metadata = f'Metadata-Version: 2.1\nName: {name}\nVersion: {version}\n'
    metadata += ''.join(f'Requires-Dist: {dep}\n' for dep in dependencies)
    with zipfile.ZipFile(directory / f'{stem}-py3-none-any.whl',
                         'w') as archive:
        archive.writestr(f'{stem}.dist-info/METADATA', metadata)
        archive.writestr(
            f'{stem}.dist-info/WHEEL',
            'Wheel-Version: 1.0\nRoot-Is-Purelib: true\n'
            'Tag: py3-none-any\n')
        archive.writestr(f'{stem}.dist-info/RECORD', '')


class LocalSdkInstallTest(unittest.TestCase):

    def test_source_sdk_wins_without_forcing_other_local_versions(self):
        rendered = subprocess.check_output([
            'make', '--no-print-directory', '-n', '-C',
            str(ROOT / 'kubernetes_platform'), 'python', '-o', 'fetch-protos',
            'USE_FIND_LINKS=true'
        ],
                                           text=True)
        command = shlex.split(rendered)[-1].split(' && ')[1]
        self.assertIn('../../sdk/python/dist/kfp-*.whl', command)
        # Run the rendered pip transaction in an isolated venv rather than a
        # Docker user site; package selection arguments remain unchanged.
        command = command.replace('--user --break-system-packages ', '')
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            local = root / 'sdk/python/dist'
            published = root / 'published'
            other_local = root / 'server-api/dist'
            wheel(local, 'kfp', '2.17.0', ['urllib3==2.8.0'])
            wheel(published, 'kfp', '2.17.0', ['urllib3==2.7.0'])
            wheel(published, 'urllib3', '2.7.0')
            wheel(published, 'urllib3', '2.8.0')
            wheel(published, 'kfp_server_api', '2.17.0')
            wheel(other_local, 'kfp_server_api', '2.17.2')
            working = root / 'kubernetes_platform/python'
            working.mkdir(parents=True)
            (working / 'requirements.txt').write_text(
                'kfp==2.17.0\nurllib3==2.8.0\nkfp-server-api==2.17.0\n')
            environment = root / 'venv'
            venv.create(environment, with_pip=True, symlinks=True)
            env = dict(
                os.environ,
                PATH=f'{environment}/bin:' + os.environ['PATH'],
                PIP_NO_INDEX='1',
                PIP_DISABLE_PIP_VERSION_CHECK='1',
                PIP_FIND_LINKS=f'{published} {local} {other_local}')
            subprocess.run(
                command,
                shell=True,
                cwd=working,
                env=env,
                check=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                text=True)
            result = subprocess.check_output([
                str(environment / 'bin/python'), '-c',
                'import importlib.metadata as m,json; '
                'print(json.dumps([m.version("urllib3"), '
                'm.version("kfp-server-api"), '
                'json.loads(m.distribution("kfp").read_text("direct_url.json"))'
                '["url"]]))'
            ],
                                             text=True)
            urllib3_version, server_version, source = json.loads(result)
            self.assertEqual(urllib3_version, '2.8.0')
            self.assertEqual(server_version, '2.17.0')
            self.assertEqual(source,
                             (local /
                              'kfp-2.17.0-py3-none-any.whl').resolve().as_uri())


if __name__ == '__main__':
    unittest.main()
