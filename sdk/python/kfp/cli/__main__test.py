# Copyright 2024 The Kubeflow Authors
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
"""Tests for kfp.cli.__main__."""

import sys
import unittest
from unittest import mock

from kfp.cli import __main__


class TestMain(unittest.TestCase):

    @mock.patch('kfp.cli.__main__.cli.cli')
    def test_main_success(self, mock_cli):
        __main__.main()
        mock_cli.assert_called_once_with(obj={}, auto_envvar_prefix='KFP')

    @mock.patch('kfp.cli.__main__.cli.cli')
    @mock.patch('kfp.cli.__main__.click.echo')
    @mock.patch('kfp.cli.__main__.sys.exit')
    def test_main_exception(self, mock_exit, mock_echo, mock_cli):
        mock_cli.side_effect = Exception('test error')
        __main__.main()
        mock_echo.assert_called_once_with('test error', err=True)
        mock_exit.assert_called_once_with(1)

    @mock.patch('kfp.cli.__main__.logging.basicConfig')
    def test_main_logging_config(self, mock_basic_config):
        with mock.patch('kfp.cli.__main__.cli.cli'):
            __main__.main()
        mock_basic_config.assert_called_once_with(
            format='%(message)s', level=20)


if __name__ == '__main__':
    unittest.main()
