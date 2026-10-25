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

import importlib.util
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[3]
SPEC = importlib.util.spec_from_file_location(
    'update_tensorboard_image', ROOT / 'hack/update_tensorboard_image.py')
updater = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(updater)


class UpdateTensorboardImageTest(unittest.TestCase):

    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        for relative in updater.REQUIRED_PATHS:
            path = self.root / relative
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(
                f'"{updater.IMAGE}2.17.1"\n"example.com/custom:stable"\n')

    def test_updates_all_sources_and_present_assets_idempotently(self):
        asset = self.root / 'sdk/python/kfp/cli/viewer_assets/index.js'
        asset.parent.mkdir(parents=True)
        asset.write_text(f'const image="{updater.IMAGE}2.17.1";')
        updater.update_images(self.root, '2.18.0-rc.1')
        paths = [self.root / path for path in updater.REQUIRED_PATHS] + [asset]
        expected = {path: path.read_text() for path in paths}
        for content in expected.values():
            self.assertIn(updater.IMAGE + '2.18.0-rc.1', content)
            self.assertNotIn(updater.IMAGE + '2.17.1', content)
        self.assertIn('example.com/custom:stable', expected[paths[0]])
        updater.update_images(self.root, '2.18.0-rc.1')
        self.assertEqual(expected, {path: path.read_text() for path in paths})

    def test_missing_image_or_file_fails_before_any_write(self):
        first = self.root / updater.REQUIRED_PATHS[0]
        last = self.root / updater.REQUIRED_PATHS[-1]
        for missing_file in [False, True]:
            with self.subTest(missing_file=missing_file):
                if missing_file:
                    last.unlink()
                else:
                    last.write_text('no default image')
                with self.assertRaises(ValueError):
                    updater.update_images(self.root, '2.18.0')
                self.assertIn(updater.IMAGE + '2.17.1', first.read_text())

    def test_invalid_tags_rejected_without_writes(self):
        for tag in ['', '../2.18', '2.18:latest', '-2.18', 'a' * 129, '2.18\n']:
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                updater.update_images(self.root, tag)
        self.assertIn('2.17.1',
                      (self.root / updater.REQUIRED_PATHS[0]).read_text())

    def test_release_script_invokes_helper(self):
        script = (ROOT / 'manifests/kustomize/hack/release.sh').read_text()
        self.assertIn(
            'python3 "${DIR}/../../../hack/update_tensorboard_image.py" "$TAG_NAME"',
            script)


if __name__ == '__main__':
    unittest.main()
