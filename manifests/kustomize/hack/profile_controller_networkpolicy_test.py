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
"""Check the rendered multi-user policy and its controller pod labels."""

import json
import sys
import unittest


class ProfileControllerNetworkPolicyTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        # yq v3 emits one JSON object per YAML document.
        remaining = sys.stdin.read().strip()
        cls.resources = []
        decoder = json.JSONDecoder()
        while remaining:
            resource, end = decoder.raw_decode(remaining)
            cls.resources.append(resource)
            remaining = remaining[end:].lstrip()

    def resource(self, kind, name):
        matches = [
            resource for resource in self.resources
            if resource.get('kind') == kind and
            resource.get('metadata', {}).get('name') == name
        ]
        self.assertEqual(len(matches), 1, f'Expected one {kind}/{name}')
        return matches[0]

    def test_policy_allows_only_metacontroller_on_sync_port(self):
        name = 'kubeflow-pipelines-profile-controller'
        policy = self.resource('NetworkPolicy', name)
        controller = self.resource('Deployment', name)
        metacontroller = self.resource('StatefulSet', 'metacontroller')

        namespace = policy['metadata']['namespace']
        self.assertEqual(controller['metadata']['namespace'], namespace)
        self.assertEqual(metacontroller['metadata']['namespace'], namespace)
        self.assertEqual(
            controller['spec']['template']['metadata']['labels']['app'], name)
        self.assertEqual(
            metacontroller['spec']['template']['metadata']['labels']['app'],
            'metacontroller')

        def selector(app):
            return {
                'matchExpressions': [{
                    'key': 'app',
                    'operator': 'In',
                    'values': [app],
                }]
            }

        # Exact selectors catch additional labels injected by transformers.
        self.assertEqual(policy['spec']['podSelector'], selector(name))
        self.assertEqual(policy['spec']['policyTypes'], ['Ingress'])
        self.assertEqual(policy['spec']['ingress'], [{
            'from': [{
                'podSelector': selector('metacontroller')
            }],
            'ports': [{
                'port': 8080,
                'protocol': 'TCP'
            }],
        }])


if __name__ == '__main__':
    unittest.main()
