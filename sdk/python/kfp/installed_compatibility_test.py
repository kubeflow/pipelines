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
import unittest
from unittest import mock

_SCRIPT = (
    Path(__file__).resolve().parents[3] / '.github' / 'resources' / 'scripts' /
    'sdk-installed-smoke.py')
_SPEC = importlib.util.spec_from_file_location('sdk_installed_smoke', _SCRIPT)
_SMOKE = importlib.util.module_from_spec(_SPEC)
_SPEC.loader.exec_module(_SMOKE)


class InstalledCompatibilityTest(unittest.TestCase):

    def test_equivalent_distribution_versions(self):
        for expected, installed in (
            ('2.18.0', '2.18.0'),
            ('2.18.1-rc.1', '2.18.1rc1'),
            ('2.18.1-dev.2', '2.18.1.dev2'),
        ):
            with self.subTest(expected=expected):
                with mock.patch.object(
                        _SMOKE.importlib.metadata,
                        'version',
                        return_value=installed) as version:
                    _SMOKE.verify_versions(expected)
                self.assertEqual(version.call_count, 4)

    def test_real_version_mismatch_is_rejected(self):
        for installed in ('2.18.0', '2.18.1rc2', '2.18.1'):
            with self.subTest(installed=installed):
                with mock.patch.object(
                        _SMOKE.importlib.metadata,
                        'version',
                        return_value=installed):
                    with self.assertRaisesRegex(RuntimeError,
                                                'kfp: expected .* got'):
                        _SMOKE.verify_versions('2.18.1-rc.1')


if __name__ == '__main__':
    unittest.main()
