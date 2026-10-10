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
"""Check handoff privileges in rendered standalone and multi-user installs."""

import json
import sys
import unittest

MULTI_USER = sys.argv.pop(1) == 'multi-user'
NAME = 'ml-pipeline-schedule-writer-handoff'
ACCOUNTS = {'ml-pipeline', 'ml-pipeline-scheduledworkflow'}


class ScheduleWriterRBACTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        remaining = sys.stdin.read().strip()
        cls.resources = []
        decoder = json.JSONDecoder()
        while remaining:
            resource, end = decoder.raw_decode(remaining)
            cls.resources.append(resource)
            remaining = remaining[end:].lstrip()

    def test_handoff_is_scoped_to_multi_user(self):
        roles = [
            r for r in self.resources
            if r.get('kind') == 'Role' and r['metadata']['name'] == NAME
        ]
        bindings = [
            r for r in self.resources
            if r.get('kind') == 'RoleBinding' and r['metadata']['name'] == NAME
        ]
        self.assertEqual(len(roles), int(MULTI_USER))
        self.assertEqual(len(bindings), int(MULTI_USER))
        if not MULTI_USER:
            # Catch moving the same privilege into a differently named role.
            for binding in self.resources:
                if binding.get('kind') not in ('RoleBinding',
                                               'ClusterRoleBinding'):
                    continue
                if not any(
                        s.get('kind') == 'ServiceAccount' and
                        s.get('name') in ACCOUNTS
                        for s in binding.get('subjects', [])):
                    continue
                for role in self.resources:
                    if role.get('kind') != binding['roleRef']['kind'] or role[
                            'metadata']['name'] != binding['roleRef']['name']:
                        continue
                    for rule in role.get('rules', []):
                        self.assertFalse(
                            '' in rule.get('apiGroups', []) and
                            'pods' in rule.get('resources', []) and
                            'patch' in rule.get('verbs', []),
                            role['metadata']['name'])
            return
        role, binding = roles[0], bindings[0]
        namespace = role['metadata']['namespace']
        self.assertEqual(binding['metadata']['namespace'], namespace)
        self.assertEqual(binding['roleRef'], {
            'apiGroup': 'rbac.authorization.k8s.io',
            'kind': 'Role',
            'name': NAME
        })
        self.assertEqual(
            sorted(binding['subjects'], key=lambda s: s['name']), [{
                'kind': 'ServiceAccount',
                'name': name,
                'namespace': namespace
            } for name in sorted(ACCOUNTS)])
        self.assertEqual(role['rules'], [
            {
                'apiGroups': [''],
                'resources': ['pods'],
                'verbs': ['get', 'list', 'patch']
            },
            {
                'apiGroups': ['apps'],
                'resources': ['deployments'],
                'resourceNames':
                    ['ml-pipeline', 'ml-pipeline-scheduledworkflow'],
                'verbs': ['get']
            },
        ])


if __name__ == '__main__':
    unittest.main()
