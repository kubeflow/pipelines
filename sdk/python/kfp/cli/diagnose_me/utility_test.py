# Copyright 2019 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License,Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Tests for diagnose_me.utility."""

import unittest
from unittest.mock import MagicMock
from unittest.mock import patch

from kfp.cli.diagnose_me import utility


class UtilityTest(unittest.TestCase):

    def test_execute_command_oserror(self):
        """Testing stdout and stderr is correctly captured upon OSError."""
        err_msg = 'Testing handling of OSError'

        with patch('subprocess.run') as mock_run:
            mock_run.side_effect = MagicMock(side_effect=OSError(err_msg))
            response = utility.execute_command([])

        self.assertEqual(response.stdout, '')
        self.assertEqual(response.stderr, err_msg)
        # An OSError raised without an errno leaves return_code unset, which
        # still counts as an error.
        self.assertIsNone(response.return_code)
        self.assertTrue(response.has_error)

    def test_execute_command_stdout(self):
        """Testing stdout output is correctly captured."""
        test_string = 'test string'
        response = utility.execute_command(['echo', test_string])

        self.assertEqual(response.stdout, test_string + '\n')
        self.assertEqual(response.stderr, '')
        self.assertEqual(response.return_code, 0)
        self.assertFalse(response.has_error)

    def test_execute_command_stderr(self):
        """Testing stderr output is correctly captured."""
        response = utility.execute_command(['ls', 'not_a_real_dir'])

        self.assertEqual(response.stdout, '')
        self.assertIn('No such file', response.stderr)
        self.assertTrue(response.has_error)

    def test_parsed_output_json(self):
        """Testing json stdout is correctly parsed."""
        response = utility.ExecutorResponse(stdout='{"key":"value"}')

        self.assertEqual(response.parsed_output, {'key': 'value'})
        self.assertEqual(response.json_output, {'key': 'value'})

    def test_parsed_output_text(self):
        """Testing non-json stdout is correctly parsed."""
        response = utility.ExecutorResponse(stdout='non-json string')

        self.assertEqual(response.parsed_output, 'non-json string')
        self.assertEqual(response.json_output, 'non-json string')

    def test_has_error(self):
        """Testing has_error reflects the return code."""
        self.assertFalse(utility.ExecutorResponse(return_code=0).has_error)
        self.assertTrue(utility.ExecutorResponse(return_code=1).has_error)


if __name__ == '__main__':
    unittest.main()
