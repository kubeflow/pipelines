# Copyright 2023 The Kubeflow Authors
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
"""Tests for config.py."""
import unittest

from kfp import local
from kfp.local import config


class LocalRunnerConfigTest(unittest.TestCase):

    def setUp(self):
        config.LocalExecutionConfig.instance = None

    def test_local_runner_config_init(self):
        """Test instance attributes with one constructor call."""
        config.LocalExecutionConfig(
            pipeline_root='my/local/root',
            runner=local.SubprocessRunner(use_venv=True),
            raise_on_error=True,
        )

        instance = config.LocalExecutionConfig.instance

        self.assertEqual(instance.pipeline_root, 'my/local/root')
        self.assertEqual(instance.runner, local.SubprocessRunner(use_venv=True))
        self.assertIs(instance.raise_on_error, True)

    def test_local_runner_config_is_singleton(self):
        """Test instance attributes with multiple constructor calls."""
        config.LocalExecutionConfig(
            pipeline_root='my/local/root',
            runner=local.SubprocessRunner(),
            raise_on_error=True,
        )
        config.LocalExecutionConfig(
            pipeline_root='other/local/root',
            runner=local.SubprocessRunner(use_venv=False),
            raise_on_error=False,
        )

        instance = config.LocalExecutionConfig.instance

        self.assertEqual(instance.pipeline_root, 'other/local/root')
        self.assertEqual(instance.runner,
                         local.SubprocessRunner(use_venv=False))
        self.assertFalse(instance.raise_on_error, False)


class TestInitCalls(unittest.TestCase):

    def setUp(self):
        config.LocalExecutionConfig.instance = None

    def test_init_more_than_once(self):
        """Tests config instance attributes with one init() call."""
        local.init(
            pipeline_root='my/local/root',
            runner=local.SubprocessRunner(use_venv=True),
        )

        instance = config.LocalExecutionConfig.instance

        self.assertEqual(instance.pipeline_root, 'my/local/root')
        self.assertEqual(instance.runner, local.SubprocessRunner(use_venv=True))

    def test_init_more_than_once(self):
        """Test config instance attributes with multiple init() calls."""
        local.init(
            pipeline_root='my/local/root',
            runner=local.SubprocessRunner(),
        )
        local.init(
            pipeline_root='other/local/root',
            runner=local.SubprocessRunner(use_venv=False),
            raise_on_error=False,
        )

        instance = config.LocalExecutionConfig.instance

        self.assertEqual(instance.pipeline_root, 'other/local/root')
        self.assertEqual(instance.runner,
                         local.SubprocessRunner(use_venv=False))
        self.assertFalse(instance.raise_on_error, False)


if __name__ == '__main__':
    unittest.main()
