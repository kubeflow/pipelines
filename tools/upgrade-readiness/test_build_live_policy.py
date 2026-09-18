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
"""Controlled-fixture RBAC merging preserves retained namespace grants."""

from contextlib import redirect_stderr
from contextlib import redirect_stdout
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import build_live_policy as builder
import schedule_policy
from target_rbac import evaluate_use


def role(name='use', namespace='team'):
    return dict(
        kind='Role',
        metadata=dict(name=name, namespace=namespace),
        rules=[
            dict(
                apiGroups=[''],
                resources=['serviceaccounts'],
                verbs=['use'],
                resourceNames=['readiness-granted'])
        ])


def binding():
    return dict(
        kind='RoleBinding',
        metadata=dict(name='use', namespace='team'),
        roleRef=dict(
            apiGroup='rbac.authorization.k8s.io', kind='Role', name='use'),
        subjects=[
            dict(
                kind='ServiceAccount',
                name='ml-pipeline-scheduledworkflow',
                namespace='kubeflow')
        ])


class BuilderTests(unittest.TestCase):

    def build(self, source, candidate, **kwargs):
        return builder.build_policy(
            dict(items=source), dict(items=candidate), 'a' * 40, 'enforce',
            **kwargs)

    def test_candidate_replaces_matching_role_and_retains_tenant_binding(self):
        source = [role(), binding(), role('other')]
        replacement = role()
        replacement['rules'][0]['resourceNames'] = ['new-account']
        policy = self.build(source, [replacement])
        self.assertIs(schedule_policy.validate(policy), policy)
        self.assertEqual(len(policy['rbac']), 3)
        self.assertEqual(
            evaluate_use(
                policy['rbac'],
                policy['controller_user'], [],
                'team',
                'new-account',
                complete=True), 'allowed')
        self.assertEqual(
            evaluate_use(
                policy['rbac'],
                policy['controller_user'], [],
                'team',
                'readiness-granted',
                complete=True), 'denied')
        self.assertEqual(source[0]['rules'][0]['resourceNames'],
                         ['readiness-granted'])

    def test_candidate_namespaced_objects_default_to_fixture_namespace(self):
        candidate = role()
        del candidate['metadata']['namespace']
        policy = self.build([role()], [candidate])
        self.assertEqual(
            [obj['metadata']['namespace'] for obj in policy['rbac']],
            ['team', 'kubeflow'])
        self.assertNotIn('namespace', candidate['metadata'])
        self.assertEqual(
            self.build([], [candidate], candidate_namespace='custom')['rbac'][0]
            ['metadata']['namespace'], 'custom')

    def test_source_requires_namespace(self):
        source = role()
        del source['metadata']['namespace']
        with self.assertRaises(ValueError):
            self.build([source], [])

    def test_rejects_duplicates_within_either_snapshot(self):
        for source, candidate in [([role(), role()], []), ([], [role(),
                                                                role()])]:
            with self.subTest(source=source):
                with self.assertRaises(ValueError):
                    self.build(source, candidate)
        missing = role()
        del missing['metadata']['namespace']
        with self.assertRaises(ValueError):
            self.build([], [missing, role(namespace='kubeflow')])

    def test_rejects_malformed_object_identity(self):
        for obj in [
                None, {},
                dict(kind='Secret', metadata=dict(name='secret')),
                dict(kind='Role'),
                dict(kind='Role', metadata={}),
                role(namespace=''),
                role(namespace=7),
                dict(
                    kind='ClusterRole',
                    metadata=dict(name='cluster', namespace='team'))
        ]:
            with self.subTest(obj=obj):
                with self.assertRaises(ValueError):
                    self.build([obj], [])

    def test_cluster_scope_and_kinds_remain_distinct(self):
        cluster = role()
        cluster['kind'] = 'ClusterRole'
        del cluster['metadata']['namespace']
        cluster_binding = binding()
        cluster_binding['kind'] = 'ClusterRoleBinding'
        del cluster_binding['metadata']['namespace']
        cluster_binding['roleRef']['kind'] = 'ClusterRole'
        self.assertEqual(
            len(
                self.build([role(), binding()],
                           [cluster, cluster_binding])['rbac']), 4)

    def test_rejects_invalid_settings_or_snapshot_shape(self):
        for revision, mode in [('short', 'enforce'), ('a' * 40, 'legacy')]:
            with self.assertRaises(ValueError):
                builder.build_policy(
                    dict(items=[]), dict(items=[]), revision, mode)
        for snapshot in [[], {}, dict(items={})]:
            with self.assertRaises(ValueError):
                builder.snapshot_objects(snapshot)

    def test_rejects_oversized_snapshot_and_merged_policy(self):
        with patch.object(builder, 'MAX_ITEMS', 1):
            with self.assertRaises(ValueError):
                self.build([role(), binding()], [])
        with patch.object(builder, 'MAX_BYTES', 10):
            with self.assertRaises(ValueError):
                self.build([], [])

    def test_cli_emits_valid_policy_without_mutating_input(self):
        with tempfile.TemporaryDirectory() as directory:
            snapshot = Path(directory) / 'snapshot.json'
            snapshot.write_text(json.dumps(dict(items=[role(), binding()])))
            output = io.StringIO()
            with patch('sys.argv', [
                    'build_live_policy.py', '--source-rbac',
                    str(snapshot), '--candidate-rbac',
                    str(snapshot), '--target-revision', 'a' * 40, '--mode',
                    'enforce'
            ]), redirect_stdout(output):
                self.assertEqual(builder.main(), 0)
            policy = json.loads(output.getvalue())
            self.assertEqual(len(schedule_policy.validate(policy)['rbac']), 2)

    def test_cli_redacts_invalid_input(self):
        with tempfile.TemporaryDirectory() as directory:
            snapshot = Path(directory) / 'snapshot.json'
            snapshot.write_text('private malformed JSON')
            output, errors = io.StringIO(), io.StringIO()
            with patch('sys.argv', [
                    'build_live_policy.py', '--source-rbac',
                    str(snapshot), '--candidate-rbac',
                    str(snapshot), '--target-revision', 'a' * 40, '--mode',
                    'enforce'
            ]), redirect_stdout(output), redirect_stderr(errors):
                self.assertEqual(builder.main(), 1)
            self.assertEqual(output.getvalue(), '')
            self.assertNotIn('private', errors.getvalue())

    def test_audit_mode_and_fixture_contract(self):
        policy = builder.build_policy(
            dict(items=[]), dict(items=[]), 'b' * 40, 'audit')
        self.assertEqual(policy['mode'], 'audit')
        self.assertTrue(policy['rbac_complete'])
        self.assertTrue(policy['rbac_only'])
        self.assertTrue(policy['multi_user'])
        self.assertEqual(policy['compiled_pipeline_spec_patch'], {})
        self.assertEqual(policy['recurring_runs'], [])
        self.assertEqual(policy['experiments'], [])


if __name__ == '__main__':
    unittest.main()
