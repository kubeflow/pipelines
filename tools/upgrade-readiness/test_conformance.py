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
from pathlib import Path
import unittest
from unittest import mock

import conformance
import schedule_policy


class ConformanceTest(unittest.TestCase):

    def test_wrong_policy_revision_cannot_run_backend(self):
        with mock.patch.object(
                conformance.subprocess, 'check_output',
                return_value='a' * 40), mock.patch.object(
                    conformance.subprocess, 'run') as run:
            with self.assertRaisesRegex(ValueError, 'pinned policy source'):
                conformance.run(Path('.'))
            run.assert_not_called()

    def test_dirty_policy_checkout_cannot_run_backend(self):
        with mock.patch.object(
                conformance.subprocess,
                'check_output',
                side_effect=[
                    schedule_policy.POLICY_SOURCE, ' M backend/source.go'
                ]), mock.patch.object(conformance.subprocess, 'run') as run:
            with self.assertRaisesRegex(ValueError, 'clean backend checkout'):
                conformance.run(Path('.'))
            run.assert_not_called()


if __name__ == '__main__':
    unittest.main()
