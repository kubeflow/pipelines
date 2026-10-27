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

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
SCRIPT = ROOT / '.github/resources/scripts/stage-runtime-image-cache.sh'
FILES = ('runtime-base-images.tar', 'modelcar.tar')


class RuntimeImageCacheTest(unittest.TestCase):
    def run_stage(self, root, outcome='success', hit='true'):
        environment = os.environ.copy()
        environment.update(
            CACHE_DIRECTORY=str(root / 'cache'),
            IMAGE_DIRECTORY=str(root / 'images'),
            CACHE_OUTCOME=outcome,
            CACHE_HIT=hit,
            GITHUB_OUTPUT=str(root / 'output'),
        )
        return subprocess.run(
            ['bash', str(SCRIPT)],
            env=environment,
            capture_output=True,
            text=True,
            check=False,
        )

    def populate(self, root):
        cache = root / 'cache'
        cache.mkdir()
        for file in FILES:
            (cache / file).write_bytes(file.encode())

    def test_complete_hit_stages_only_declared_files(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            self.populate(root)
            (root / 'cache/extra').write_text('unrelated')
            result = self.run_stage(root)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual((root / 'output').read_text(), 'ready=true\n')
            self.assertEqual(
                {p.name for p in (root / 'images').iterdir()}, set(FILES)
            )
            for file in FILES:
                self.assertEqual(
                    (root / 'images' / file).read_bytes(), file.encode()
                )
            self.assertTrue((root / 'cache/extra').exists())

    def test_miss_failure_missing_and_empty_files_fall_back(self):
        cases = (
            ('success', 'false', None),
            ('failure', 'true', None),
            ('success', 'true', 'missing'),
            ('success', 'true', 'empty'),
        )
        for outcome, hit, damage in cases:
            with self.subTest(outcome=outcome, hit=hit, damage=damage):
                with tempfile.TemporaryDirectory() as directory:
                    root = Path(directory).resolve()
                    self.populate(root)
                    if damage == 'missing':
                        (root / 'cache/modelcar.tar').unlink()
                    elif damage == 'empty':
                        (root / 'cache/runtime-base-images.tar').write_bytes(
                            b''
                        )
                    result = self.run_stage(root, outcome, hit)
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(
                        (root / 'output').read_text(), 'ready=false\n'
                    )
                    self.assertFalse((root / 'images').exists())
                    self.assertFalse(
                        any((root / 'cache' / file).exists() for file in FILES)
                    )

    def test_invalid_source_or_destination_is_fatal(self):
        for source in (True, False):
            with self.subTest(source=source):
                with tempfile.TemporaryDirectory() as directory:
                    root = Path(directory).resolve()
                    self.populate(root)
                    target = root / 'target'
                    target.write_bytes(b'preserved')
                    if source:
                        path = root / 'cache/modelcar.tar'
                        path.unlink()
                    else:
                        (root / 'images').mkdir()
                        path = root / 'images/modelcar.tar'
                    path.symlink_to(target)
                    result = self.run_stage(root)
                    self.assertEqual(result.returncode, 2, result.stdout)
                    self.assertFalse((root / 'output').exists())
                    self.assertEqual(target.read_bytes(), b'preserved')

    def test_cache_contract_matches_producer(self):
        action = yaml.safe_load(
            (ROOT / '.github/actions/deploy/action.yml').read_text()
        )
        steps = action['runs']['steps']
        restore = next(
            step for step in steps if step.get('id') == 'runtime-image-cache'
        )
        configure = next(
            step
            for step in steps
            if step.get('id') == 'runtime-image-cache-key'
        )
        producer = yaml.safe_load(
            (ROOT / '.github/workflows/runtime-base-images.yml').read_text()
        )['jobs']['produce-runtime-base-images']
        producer_restore = next(
            step
            for step in producer['steps']
            if step.get('id') == 'cache-runtime-base-images'
        )
        producer_paths = producer_restore['with']['path']
        for name, value in producer['env'].items():
            producer_paths = producer_paths.replace(
                '${{ env.' + name + ' }}', value
            )
        self.assertEqual(restore['with']['path'], producer_paths)
        self.assertEqual(restore['uses'], producer_restore['uses'])
        self.assertNotIn('restore-keys', restore['with'])
        producer_configure = next(
            step
            for step in producer['steps']
            if step.get('id') == 'configure-runtime-base-images'
        )
        identity_command = 'python3 .github/resources/scripts/runtime_base_image_artifacts.py identity >> "$GITHUB_OUTPUT"'
        self.assertIn(identity_command, configure['run'])
        self.assertIn(identity_command, producer_configure['run'])
        stage = next(
            step
            for step in steps
            if step.get('id') == 'runtime-image-cache-files'
        )
        self.assertEqual(
            stage['env']['IMAGE_DIRECTORY'],
            '${{ inputs.image_path }}/runtime-base-images',
        )
        self.assertFalse(
            any('actions/cache/save@' in step.get('uses', '') for step in steps)
        )


if __name__ == '__main__':
    unittest.main()
