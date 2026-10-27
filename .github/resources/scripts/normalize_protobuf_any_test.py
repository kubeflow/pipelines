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

import copy
import json
from pathlib import Path
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[3]
FILTER = ROOT / 'backend/api/hack/normalize-protobuf-any.jq'


def normalize(schema):
    result = subprocess.run(['jq', '-f', str(FILTER)],
                            input=json.dumps(schema),
                            capture_output=True,
                            text=True,
                            check=True)
    return json.loads(result.stdout)


class NormalizeProtobufAnyTest(unittest.TestCase):

    def test_new_schema_restores_existing_client_contract(self):
        for version in ('v1beta1', 'v2beta1'):
            with self.subTest(version=version):
                path = ROOT / 'backend/api' / version / 'swagger/run.swagger.json'
                original = json.loads(path.read_text())
                updated = copy.deepcopy(original)
                updated['definitions']['protobufAny'][
                    'additionalProperties'] = True
                self.assertEqual(normalize(updated), original)
                self.assertEqual(normalize(normalize(updated)), original)

    def test_other_schemas_are_unchanged(self):
        for schema in ({}, {
                'definitions': {}
        }, {
                'definitions': {
                    'Other': {
                        'additionalProperties': True
                    },
                    'protobufAny': {
                        'additionalProperties': False
                    },
                }
        }, {
                'definitions': {
                    'protobufAny': {
                        'additionalProperties': {
                            'type': 'string'
                        }
                    }
                }
        }):
            with self.subTest(schema=schema):
                self.assertEqual(normalize(schema), schema)


if __name__ == '__main__':
    unittest.main()
