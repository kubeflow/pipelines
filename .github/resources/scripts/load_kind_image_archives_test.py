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

import json
import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest

SCRIPT = Path(__file__).with_name('load-kind-image-archives.sh')


class LoadKindImageArchivesTest(unittest.TestCase):

    def _run(
        self,
        archive_names: tuple[str, ...],
        *,
        concurrency: str = '2',
        fail_archive: str = '',
    ) -> tuple[subprocess.CompletedProcess[str], Path, dict[str, object]]:
        temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(temporary_directory.cleanup)
        root = Path(temporary_directory.name)
        fake_bin = root / 'bin'
        fake_bin.mkdir()
        state_path = root / 'state.json'
        state_path.write_text(
            json.dumps({
                'active': 0,
                'maximum': 0,
                'archives': []
            }),
            encoding='utf-8',
        )

        fake_kind = fake_bin / 'kind'
        fake_kind.write_text(
            textwrap.dedent('''\
                #!/usr/bin/env python3
                import fcntl
                import json
                import os
                from pathlib import Path
                import sys
                import time

                archive = Path(sys.argv[-1]).name
                state_path = Path(os.environ['KIND_STATE'])
                with state_path.open('r+', encoding='utf-8') as state_file:
                    fcntl.flock(state_file, fcntl.LOCK_EX)
                    state = json.load(state_file)
                    state['active'] += 1
                    state['maximum'] = max(state['maximum'], state['active'])
                    state['archives'].append(archive)
                    state_file.seek(0)
                    json.dump(state, state_file)
                    state_file.truncate()
                time.sleep(0.08)
                with state_path.open('r+', encoding='utf-8') as state_file:
                    fcntl.flock(state_file, fcntl.LOCK_EX)
                    state = json.load(state_file)
                    state['active'] -= 1
                    state_file.seek(0)
                    json.dump(state, state_file)
                    state_file.truncate()
                if archive == os.environ.get('FAIL_ARCHIVE'):
                    sys.exit(42)
            '''),
            encoding='utf-8',
        )
        fake_kind.chmod(0o755)

        archives = []
        for archive_name in archive_names:
            archive = root / archive_name
            archive.write_text('archive', encoding='utf-8')
            archives.append(archive)

        environment = os.environ.copy()
        environment.update({
            'FAIL_ARCHIVE': fail_archive,
            'KIND_STATE': str(state_path),
            'PATH': f'{fake_bin}{os.pathsep}{environment["PATH"]}',
        })
        result = subprocess.run(
            [
                'bash',
                str(SCRIPT),
                'test-cluster',
                concurrency,
                *(str(archive) for archive in archives),
            ],
            capture_output=True,
            text=True,
            check=False,
            env=environment,
        )
        state = json.loads(state_path.read_text(encoding='utf-8'))
        return result, root, state

    def test_loads_archives_with_bounded_parallelism(self):
        names = ('one.tar', 'two.tar', 'three.tar', 'four.tar')
        result, root, state = self._run(names)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(state['maximum'], 2)
        self.assertCountEqual(state['archives'], names)
        for name in names:
            self.assertFalse((root / name).exists())

    def test_preserves_failed_archive_and_returns_failure(self):
        result, root, state = self._run(('one.tar', 'two.tar'),
                                        fail_archive='two.tar')

        self.assertEqual(result.returncode, 1)
        self.assertIn('Failed to load Kind image archive', result.stderr)
        self.assertFalse((root / 'one.tar').exists())
        self.assertTrue((root / 'two.tar').exists())
        self.assertEqual(state['active'], 0)

    def test_rejects_invalid_parallelism(self):
        result, root, state = self._run(('one.tar',), concurrency='0')

        self.assertEqual(result.returncode, 2)
        self.assertIn('MAX_PARALLEL must be a positive integer', result.stderr)
        self.assertTrue((root / 'one.tar').exists())
        self.assertEqual(state['archives'], [])


if __name__ == '__main__':
    unittest.main()
