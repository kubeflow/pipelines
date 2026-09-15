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

import argparse
import copy
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

from deduplicate_sarif import deduplicate_sarif
from deduplicate_sarif import main
from deduplicate_sarif import write_sarif


def create_result() -> dict:
    return {
        'ruleId': 'CVE-2026-1234',
        'level': 'warning',
        'message': {
            'text': "Package 'example@1.0' is vulnerable"
        },
        'locations': [{
            'physicalLocation': {
                'artifactLocation': {
                    'uri': 'file:///packages/example/METADATA'
                }
            }
        }],
        'partialFingerprints': {
            'primaryLocationLineHash': 'abc123'
        },
    }


class DeduplicateSarifTest(unittest.TestCase):

    def test_removes_only_exact_duplicates_and_preserves_order(self):
        first_result = create_result()
        second_result = copy.deepcopy(first_result)
        distinct_result = create_result()
        distinct_result['ruleId'] = 'CVE-2026-5678'
        sarif = {
            'runs': [{
                'results': [first_result, second_result, distinct_result]
            }]
        }

        self.assertEqual(deduplicate_sarif(sarif), 1)
        self.assertEqual(sarif['runs'][0]['results'],
                         [first_result, distinct_result])

    def test_preserves_any_result_metadata_difference(self):
        base_result = create_result()
        variants = []
        for field, value in (
            ('ruleId', 'CVE-2026-9999'),
            ('level', 'error'),
            ('message', {
                'text': 'A different package is vulnerable'
            }),
            ('locations', []),
            ('partialFingerprints', {
                'primaryLocationLineHash': 'different-layer'
            }),
        ):
            variant = copy.deepcopy(base_result)
            variant[field] = value
            variants.append(variant)
        sarif = {'runs': [{'results': [base_result, *variants]}]}

        self.assertEqual(deduplicate_sarif(sarif), 0)
        self.assertEqual(len(sarif['runs'][0]['results']), 6)

    def test_deduplicates_each_run_independently(self):
        result = create_result()
        sarif = {
            'runs': [
                {
                    'results': [result, copy.deepcopy(result)]
                },
                {
                    'results': [copy.deepcopy(result)]
                },
                {
                    'tool': {
                        'driver': {
                            'name': 'empty'
                        }
                    }
                },
            ]
        }

        self.assertEqual(deduplicate_sarif(sarif), 1)
        self.assertEqual(len(sarif['runs'][0]['results']), 1)
        self.assertEqual(len(sarif['runs'][1]['results']), 1)
        self.assertNotIn('results', sarif['runs'][2])

    def test_rejects_malformed_sarif(self):
        invalid_documents = (
            {},
            {'runs': {}},
            {'runs': ['invalid-run']},
            {'runs': [{'results': {}}]},
            {'runs': [{'results': ['invalid-result']}]},
        )

        for invalid_document in invalid_documents:
            with self.subTest(invalid_document=invalid_document):
                with self.assertRaises(ValueError):
                    deduplicate_sarif(invalid_document)

    def test_write_sarif_creates_parseable_output(self):
        sarif = {'version': '2.1.0', 'runs': []}
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_path = Path(temporary_directory) / 'results.sarif'

            write_sarif(output_path, sarif)

            self.assertEqual(json.loads(output_path.read_text()), sarif)

    @mock.patch('deduplicate_sarif.parse_args')
    def test_main_does_not_replace_output_for_invalid_input(self,
                                                            parse_args_mock):
        with tempfile.TemporaryDirectory() as temporary_directory:
            input_path = Path(temporary_directory) / 'invalid.sarif'
            output_path = Path(temporary_directory) / 'results.sarif'
            input_path.write_text('not JSON', encoding='utf-8')
            output_path.write_text('existing output', encoding='utf-8')
            parse_args_mock.return_value = argparse.Namespace(
                input=input_path, output=output_path)

            with self.assertRaises(RuntimeError):
                main()

            self.assertEqual(output_path.read_text(encoding='utf-8'),
                             'existing output')


if __name__ == '__main__':
    unittest.main()
