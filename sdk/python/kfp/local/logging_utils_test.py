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
"""Tests for logging_utils.py."""

import io
import unittest
from unittest import mock

from kfp import dsl
from kfp.local import logging_utils
from kfp.local import status


class TestIndentedPrint(unittest.TestCase):

    @mock.patch('sys.stdout', new_callable=io.StringIO)
    def test(self, mocked_stdout):
        with logging_utils.indented_print(num_spaces=6):
            print('foo should be indented')
        expected = '      foo should be indented\n'
        actual = mocked_stdout.getvalue()
        self.assertEqual(
            actual,
            expected,
        )


class TestColorText(unittest.TestCase):

    def test_cyan(self):

        actual = logging_utils.color_text(
            'text to color',
            logging_utils.Color.CYAN,
        )
        expected = '\x1b[91mtext to color\x1b[0m'
        self.assertEqual(actual, expected)

    def test_cyan(self):

        actual = logging_utils.color_text(
            'text to color',
            logging_utils.Color.RED,
        )
        expected = '\x1b[91mtext to color\x1b[0m'
        self.assertEqual(actual, expected)


class TestRenderArtifact(unittest.TestCase):

    def test_empty(self):
        actual = logging_utils.make_log_lines_for_artifact(dsl.Artifact())
        expected = [
            "Artifact( name='',",
            "          uri='',",
            '          metadata={} )',
        ]
        self.assertListEqual(actual, expected)

    def test_contains_value(self):
        actual = logging_utils.make_log_lines_for_artifact(
            dsl.Model(
                name='my_artifact',
                uri='/local/foo/bar',
                metadata={
                    'dict_field': {
                        'baz': 'bat'
                    },
                    'float_field': 3.14
                }))
        expected = [
            "Model( name='my_artifact',",
            "       uri='/local/foo/bar',",
            "       metadata={'dict_field': {'baz': 'bat'}, 'float_field': 3.14} )",
        ]
        self.assertListEqual(actual, expected)


class TestMakeLogLinesForOutputs(unittest.TestCase):

    def test_empty(self):
        actual = logging_utils.make_log_lines_for_outputs(dict())
        expected = []
        self.assertListEqual(actual, expected)

    def test_only_params(self):
        actual = logging_utils.make_log_lines_for_outputs({
            'foo': 'bar',
            'baz': 100,
            'bat': 1.0,
            'brap': True,
            'my_list': [1, 2, 3],
            'my_dict': {
                'foo': 'bar'
            }
        })
        expected = [
            "    foo: 'bar'",
            '    baz: 100',
            '    bat: 1.0',
            '    brap: True',
            '    my_list: [1, 2, 3]',
            "    my_dict: {'foo': 'bar'}",
        ]
        self.assertListEqual(actual, expected)

    def test_only_artifacts(self):
        actual = logging_utils.make_log_lines_for_outputs({
            'my_artifact':
                dsl.Artifact(name=''),
            'my_model':
                dsl.Model(
                    name='my_artifact',
                    uri='/local/foo/bar/1234567890/1234567890/1234567890/1234567890/1234567890',
                    metadata={
                        'dict_field': {
                            'baz': 'bat'
                        },
                        'float_field': 3.14
                    }),
            'my_dataset':
                dsl.Dataset(
                    name='my_dataset',
                    uri='/local/foo/baz',
                    metadata={},
                ),
        })
        expected = [
            "    my_artifact: Artifact( name='',",
            "                           uri='',",
            '                           metadata={} )',
            "    my_model: Model( name='my_artifact',",
            "                     uri='/local/foo/bar/1234567890/1234567890/1234567890/1234567890/1234567890',",
            "                     metadata={'dict_field': {'baz': 'bat'}, 'float_field': 3.14} )",
            "    my_dataset: Dataset( name='my_dataset',",
            "                         uri='/local/foo/baz',",
            '                         metadata={} )',
        ]
        self.assertListEqual(actual, expected)

    def test_mix_params_and_artifacts(self):
        actual = logging_utils.make_log_lines_for_outputs({
            'foo':
                'bar',
            'baz':
                100,
            'bat':
                1.0,
            'brap':
                True,
            'my_list': [1, 2, 3],
            'my_dict': {
                'foo': 'bar'
            },
            'my_artifact':
                dsl.Artifact(name=''),
            'my_model':
                dsl.Model(
                    name='my_artifact',
                    uri='/local/foo/bar/1234567890/1234567890/1234567890/1234567890/1234567890',
                    metadata={
                        'dict_field': {
                            'baz': 'bat'
                        },
                        'float_field': 3.14
                    }),
            'my_dataset':
                dsl.Dataset(
                    name='my_dataset',
                    uri='/local/foo/baz',
                    metadata={},
                ),
        })
        expected = [
            "    foo: 'bar'",
            '    baz: 100',
            '    bat: 1.0',
            '    brap: True',
            '    my_list: [1, 2, 3]',
            "    my_dict: {'foo': 'bar'}",
            "    my_artifact: Artifact( name='',",
            "                           uri='',",
            '                           metadata={} )',
            "    my_model: Model( name='my_artifact',",
            "                     uri='/local/foo/bar/1234567890/1234567890/1234567890/1234567890/1234567890',",
            "                     metadata={'dict_field': {'baz': 'bat'}, 'float_field': 3.14} )",
            "    my_dataset: Dataset( name='my_dataset',",
            "                         uri='/local/foo/baz',",
            '                         metadata={} )',
        ]

        self.assertListEqual(actual, expected)


class TestFormatStatus(unittest.TestCase):

    def test_success_status(self):
        self.assertEqual(
            logging_utils.format_status(status.Status.SUCCESS),
            '\x1b[92mSUCCESS\x1b[0m')

    def test_failure_status(self):
        self.assertEqual(
            logging_utils.format_status(status.Status.FAILURE),
            '\x1b[91mFAILURE\x1b[0m')

    def test_invalid_status(self):
        with self.assertRaisesRegex(ValueError,
                                    r'Got unknown status: INVALID_STATUS'):
            logging_utils.format_status('INVALID_STATUS')


class TestFormatTaskName(unittest.TestCase):

    def test(self):
        self.assertEqual(
            logging_utils.format_task_name('my-task'),
            '\x1b[96m\'my-task\'\x1b[0m')


class TestFormatPipelineName(unittest.TestCase):

    def test(self):
        self.assertEqual(
            logging_utils.format_pipeline_name('my-pipeline'),
            '\033[95m\'my-pipeline\'\033[0m')


if __name__ == '__main__':
    unittest.main()
