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
"""Keep the legacy cache fixture aligned with CI's preloaded runtime images."""

import ast
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[3]


class RuntimeFixtureImagesTest(unittest.TestCase):
    def test_proxy_fixture_images_are_preloaded(self):
        inventory = set((ROOT / '.github/resources/runtime-base-images.txt')
                        .read_text().splitlines())
        fixture = (ROOT / 'test_data/sdk_compiled_pipelines/valid/env-var.yaml').read_text()
        images = re.findall(r'^\s+image: (\S+)\s*$', fixture, re.MULTILINE)
        self.assertTrue(images)
        self.assertTrue(set(images) <= inventory,
                        f'Images missing from preload inventory: {set(images) - inventory}')
        source = ast.parse((ROOT / 'test_data/sdk_compiled_pipelines/valid/env_var.py').read_text())
        source_images = [ast.literal_eval(keyword.value)
                         for node in ast.walk(source)
                         if isinstance(node, ast.Call)
                         for keyword in node.keywords
                         if keyword.arg == 'base_image']
        self.assertEqual(source_images, images)
        backend_fixture = (ROOT / 'backend/test/v2/resources/env-var.yaml').read_text()
        self.assertEqual(re.findall(r'^\s+image: (\S+)\s*$', backend_fixture,
                                    re.MULTILINE), images)

    def test_legacy_cache_fixture_images_are_preloaded(self):
        inventory = set((ROOT / '.github/resources/runtime-base-images.txt')
                        .read_text().splitlines())
        fixture = (ROOT / 'backend/test/v2/resources/'
                   'hello-world-with-returning-component.yaml').read_text()
        images = re.findall(r'^\s+image: (\S+)\s*$', fixture, re.MULTILINE)
        self.assertTrue(images, 'fixture must contain a container image')
        self.assertTrue(set(images) <= inventory,
                        f'Images missing from preload inventory: {set(images) - inventory}')
        source = ast.parse((ROOT / 'test_data/sdk_uncompiled_pipelines/'
                            'hello-world-with-returning-component.py').read_text())
        source_images = [ast.literal_eval(keyword.value)
                         for node in ast.walk(source)
                         if isinstance(node, ast.Call)
                         for keyword in node.keywords
                         if keyword.arg == 'base_image']
        self.assertEqual(source_images, images)


if __name__ == '__main__':
    unittest.main()
