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

import copy
import unittest

from target_rbac import evaluate_use

GROUP = 'rbac.authorization.k8s.io'
USER = 'system:serviceaccount:system:controller'
GROUPS = [
    'system:serviceaccounts', 'system:serviceaccounts:system',
    'system:authenticated'
]


def evidence():
    return [
        {
            'kind':
                'Role',
            'metadata': {
                'name': 'submit',
                'namespace': 'team'
            },
            'rules': [{
                'apiGroups': [''],
                'resources': ['serviceaccounts'],
                'verbs': ['use'],
                'resourceNames': ['runner']
            }]
        },
        {
            'kind':
                'RoleBinding',
            'metadata': {
                'name': 'submit',
                'namespace': 'team'
            },
            'roleRef': {
                'apiGroup': GROUP,
                'kind': 'Role',
                'name': 'submit'
            },
            'subjects': [{
                'kind': 'ServiceAccount',
                'name': 'controller',
                'namespace': 'system'
            }]
        },
    ]


def check(items, complete=True, **kwargs):
    args = dict(
        user=USER,
        groups=GROUPS,
        namespace='team',
        account='runner',
        complete=complete)
    args.update(kwargs)
    return evaluate_use(items, **args)


class TargetRbacTest(unittest.TestCase):

    def test_named_grant_and_partial_denial(self):
        self.assertEqual(check(evidence(), complete=False), 'allowed')
        self.assertEqual(check(evidence(), account='other'), 'denied')
        self.assertEqual(check(evidence(), False, account='other'), 'unknown')
        self.assertEqual(check([]), 'denied')
        self.assertEqual(check([], False), 'unknown')

    def test_wildcard_rules_but_literal_resource_names(self):
        items = evidence()
        rule = items[0]['rules'][0]
        rule.update(apiGroups=['*'], verbs=['*'], resources=['*'])
        self.assertEqual(check(items), 'allowed')
        rule['resourceNames'] = ['*']
        self.assertEqual(check(items), 'denied')
        rule['resourceNames'] = []
        self.assertEqual(check(items), 'allowed')
        del rule['resourceNames']
        self.assertEqual(check(items), 'allowed')
        rule['resources'] = ['serviceaccounts/*']
        self.assertEqual(check(items), 'denied')

    def test_api_group_and_verb_must_match(self):
        for field, value in [('apiGroups', ['pipelines.kubeflow.org']),
                             ('verbs', ['get']), ('resources', ['pods'])]:
            items = evidence()
            items[0]['rules'][0][field] = value
            self.assertEqual(check(items), 'denied')

    def test_user_and_group_subjects(self):
        for kind, name in [('User', USER), ('Group', GROUPS[1])]:
            items = evidence()
            items[1]['subjects'] = [dict(kind=kind, name=name, apiGroup=GROUP)]
            self.assertEqual(check(items), 'allowed')
            items[1]['subjects'][0]['name'] = 'another'
            self.assertEqual(check(items), 'denied')

    def test_serviceaccount_namespace_default_and_scope(self):
        items = evidence()
        del items[1]['subjects'][0]['namespace']
        self.assertEqual(check(items), 'denied')
        self.assertEqual(
            check(items, user='system:serviceaccount:team:controller'),
            'allowed')
        items[1]['metadata']['namespace'] = 'another'
        self.assertEqual(check(items), 'denied')

    def test_clusterrole_bindings(self):
        items = evidence()
        items[0]['kind'] = 'ClusterRole'
        del items[0]['metadata']['namespace']
        items[1]['roleRef']['kind'] = 'ClusterRole'
        self.assertEqual(check(items), 'allowed')
        items[1]['kind'] = 'ClusterRoleBinding'
        del items[1]['metadata']['namespace']
        self.assertEqual(check(items, namespace='another'), 'allowed')
        del items[1]['subjects'][0]['namespace']
        self.assertEqual(check(items), 'unknown')
        items[1]['roleRef']['kind'] = 'Role'
        self.assertEqual(check(items), 'unknown')

    def test_nonempty_aggregated_rules_are_unresolved(self):
        for verbs in (['use'], ['get']):
            items = evidence()
            items[0]['kind'] = 'ClusterRole'
            items[0]['metadata'].pop('namespace')
            items[0]['aggregationRule'] = {'clusterRoleSelectors': [{}]}
            items[0]['rules'][0]['verbs'] = verbs
            items[1]['roleRef']['kind'] = 'ClusterRole'
            self.assertEqual(check(items), 'unknown')
            self.assertEqual(check(items + evidence()), 'allowed')

    def test_missing_duplicate_and_aggregated_roles(self):
        items = evidence()
        self.assertEqual(check(items[1:]), 'unknown')
        self.assertEqual(check(items + [copy.deepcopy(items[0])]), 'unknown')
        items[0]['aggregationRule'] = {'clusterRoleSelectors': [{}]}
        items[0]['rules'] = []
        self.assertEqual(check(items), 'unknown')
        del items[0]['rules']
        self.assertEqual(check(items), 'unknown')

    def test_malformed_rules_never_grant(self):
        for field in ('apiGroups', 'verbs', 'resources', 'resourceNames'):
            for value in ('*', [42], None, {}):
                items = evidence()
                items[0]['rules'][0][field] = value
                self.assertEqual(check(items), 'unknown', (field, value))
        for rules in ('*', {}, [None], [{'verbs': ['use']}], None):
            items = evidence()
            items[0]['rules'] = rules
            self.assertEqual(check(items), 'unknown')

    def test_malformed_subject_or_reference(self):
        for subject in (None, {}, {
                'kind': 'Group',
                'name': GROUPS[0]
        }, {
                'kind': 'ServiceAccount',
                'name': 'controller',
                'namespace': 1
        }, {
                'kind': 'Other',
                'name': USER
        }):
            items = evidence()
            items[1]['subjects'] = [subject]
            self.assertEqual(check(items), 'unknown')
        for ref in (None, {}, {'kind': 'Role', 'name': 'submit'}):
            items = evidence()
            items[1]['roleRef'] = ref
            self.assertEqual(check(items), 'unknown')

    def test_unrelated_binding_does_not_poison_evidence(self):
        items = evidence()
        items[1]['subjects'][0]['name'] = 'another'
        items[1]['roleRef'] = None
        self.assertEqual(check(items), 'denied')
        items[1]['metadata']['namespace'] = 'another'
        items[1]['subjects'] = 'malformed'
        self.assertEqual(check(items), 'denied')

    def test_additive_grant_overrides_unknown(self):
        items = evidence()
        broken = copy.deepcopy(items[1])
        broken['roleRef']['name'] = 'missing'
        self.assertEqual(check([broken] + items), 'allowed')
        items[0]['rules'].insert(0, None)
        self.assertEqual(check(items), 'allowed')

    def test_non_resource_rules_do_not_grant(self):
        items = evidence()
        items[0]['rules'] = [{'verbs': ['*'], 'nonResourceURLs': ['*']}]
        self.assertEqual(check(items), 'denied')

    def test_identity_input_shape(self):
        for changes in ({
                'groups': '*'
        }, {
                'user': ''
        }, {
                'namespace': None
        }, {
                'account': 4
        }, {
                'complete': 'true'
        }):
            self.assertEqual(check(evidence(), **changes), 'unknown')


if __name__ == '__main__':
    unittest.main()
