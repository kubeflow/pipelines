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
import json
import unittest
from unittest import mock

import workload_assessment as assessment
from workload_inventory import RESOURCES


def inventory(record):
    data = {key: [] for key in RESOURCES}
    data['runs'] = [dict(run_id='run-1', namespace='team-a', **record)]
    return data


def v2(**kwargs):
    return dict(
        pipelineInfo={'name': 'sensitive-pipeline-name'}, root={}, **kwargs)


class AssessmentTest(unittest.TestCase):

    def test_template_compatibility_depends_on_runtime(self):
        legacy = dict(
            kind='Workflow', apiVersion='argoproj.io/v1alpha1', spec={})
        for runtime, expected in [('v2-only', 'policy_rejection'),
                                  ('legacy-and-v2', 'unknown'),
                                  (None, 'unknown')]:
            result, summary = assessment.assess(
                inventory(dict(pipeline_spec=legacy)), dict(runtime=runtime))
            self.assertEqual(summary['legacy_argo'], 1)
            self.assertEqual([
                r['status'] for r in result if r['rule'] == 'workload.template'
            ], [expected])
            self.assertTrue(any(r['rule'] == 'workload.cache' for r in result))

    def test_stored_json_never_infers_original_boundary_sizes(self):
        result, summary = assessment.assess(
            inventory(
                dict(
                    pipelineSpec=v2(),
                    runtimeConfig={'parameters': {
                        'secret': 'never-print'
                    }})), dict(limits={
                        'spec_bytes': 1,
                        'upload_bytes': 1
                    }))
        self.assertEqual(summary['v2_ir'], 1)
        self.assertTrue(
            any(r['rule'] == 'workload.size' and r['status'] == 'unknown'
                for r in result))
        self.assertFalse(
            any(r['rule'].startswith('workload.size.') for r in result))
        self.assertNotIn('never-print', json.dumps(result))
        self.assertNotIn('sensitive-pipeline-name', json.dumps(result))

    def test_independent_measured_size_boundaries_and_exact_limit(self):
        target = dict(
            limits={name: 100 for name in assessment.SIZE_FIELDS},
            resource_evidence=[
                dict(
                    resource_kind='run',
                    resource_id='run-1',
                    upload_bytes=101,
                    spec_bytes=100,
                    metrics_bytes=0)
            ])
        result, _ = assessment.assess(
            inventory(dict(pipeline_spec=v2())), target)
        checks = {r['rule']: r['status'] for r in result}
        self.assertEqual(checks['workload.size.upload_bytes'],
                         'policy_rejection')
        self.assertEqual(checks['workload.size.spec_bytes'], 'unknown')
        self.assertEqual(checks['workload.size.metrics_bytes'], 'unknown')

    def test_cross_namespace_private_reference_and_shared_exception(self):
        data = inventory(
            dict(
                _readiness_version_reference=dict(
                    kind='pinned', resolution='observed', pipeline_id='p')))
        for namespace, shared_read, expected in [('team-b', False,
                                                  'policy_rejection'),
                                                 ('', False, 'unknown'),
                                                 ('-', False, 'unknown'),
                                                 ('team-a', False, 'unknown'),
                                                 ('team-b', True, 'unknown')]:
            data['pipelines'] = [dict(pipeline_id='p', namespace=namespace)]
            result, _ = assessment.assess(
                data, dict(multi_user=True, shared_read=shared_read))
            self.assertEqual([
                r['status'] for r in result if r['rule'] == 'workload.reference'
            ], [expected])

    def test_http_origin_path_credential_redaction_and_gateway_unknown(self):
        examples = [
            ('https://files.test/approved/object?token=secret-query',
             'https://files.test/approved/', 'unknown'),
            ('https://files.test/approved-evil/object',
             'https://files.test/approved/', 'policy_rejection'),
            ('https://other.test/approved/object',
             'https://files.test/approved/', 'policy_rejection'),
            ('https://files.test:443/approved/object',
             'https://files.test/approved/', 'unknown'),
            ('https://files.test/approved/object',
             ' https://files.test/approved/ ', 'unknown'),
            ('https://%66iles.test/approved/object',
             'https://files.test/approved/', 'unknown'),
            ('http://127.1/approved/object', 'http://127.0.0.1/approved/',
             'unknown'),
            ('https://files.test/root/object',
             'https://files.test/approved/?secret=1', 'unknown'),
            ('https://files.test/a/%2e%2e/approved/object',
             'https://files.test/approved/', 'unknown'),
            ('https://user:secret-password@files.test/approved/object',
             'https://files.test/approved/', 'unknown'),
            ('https://files.test/approved/object', 'gateway.test/artifacts/',
             'unknown'),
            ('https://files.test/approved/object', '', 'policy_rejection'),
        ]
        for uri, base, expected in examples:
            with self.subTest(uri=uri, base=base):
                result, _ = assessment.assess(
                    inventory(dict(runtime_config={'pipeline_root': uri})),
                    dict(http_base_url=base))
                artifact = [
                    r for r in result if r['rule'] == 'workload.artifacts'
                ]
                self.assertEqual([r['status'] for r in artifact], [expected])
                serialized = json.dumps(result)
                for secret in ('secret-query', 'secret-password',
                               'approved/object'):
                    self.assertNotIn(secret, serialized)

    def test_source_roots_and_mlmd_artifact_ids_are_distinct(self):
        record = dict(
            pipeline_spec=v2(
                defaultPipelineRoot='s3://secret-bucket/secret-key'),
            runtimeConfig={
                'pipelineRoot': 'https://files.test/root/key?private-value'
            },
            runDetails={
                'taskDetails': [{
                    'outputs': {
                        'dataset': {
                            'artifactIds': ['123', '456']
                        }
                    }
                }]
            })
        data = inventory(record)
        original = copy.deepcopy(data)
        result, _ = assessment.assess(data)
        self.assertEqual(data, original)
        self.assertEqual(
            len([r for r in result if r['rule'] == 'workload.artifacts']), 2)
        self.assertTrue(
            any(r['rule'] == 'workload.artifactCoverage' and
                'MLMD IDs' in r['evidence'] for r in result))
        for secret in ('secret-bucket', 'secret-key', 'private-value'):
            self.assertNotIn(secret, json.dumps(result))

    def test_unresolved_annotations_produce_unknown_checks(self):
        result, _ = assessment.assess(
            inventory(
                dict(
                    _readiness_collection_errors=['pipeline_not_available'],
                    _readiness_version_reference={
                        'kind': 'moving_latest',
                        'resolution': 'unresolved'
                    })))
        self.assertTrue(any(r['rule'] == 'workload.collection' for r in result))
        self.assertTrue(
            all(r['coverage'] == 'partial'
                for r in assessment.coverage(result)))

    def test_target_validation_rejects_invalid_or_ambiguous_measurements(self):
        bad_targets = [
            dict(runtime='master'),
            dict(limits={'upload_bytes': True}),
            dict(limits={'spec_bytes': 134217729}),
            dict(limits={'custom': 1}),
            dict(http_base_url=1),
            dict(resource_evidence=[
                dict(resource_kind='run', resource_id='r', metrics_bytes=-1)
            ]),
            dict(
                resource_evidence=[dict(resource_kind='run', resource_id='r')] *
                2)
        ]
        for target in bad_targets:
            with self.subTest(target=target), self.assertRaises(ValueError):
                assessment.validate_target(target)

    def test_finding_budget_retains_evidence_and_reports_remaining_scope(self):
        data = inventory(dict(pipeline_spec=v2()))
        data['runs'].append(
            dict(run_id='run-2', namespace='team-a', pipeline_spec=v2()))
        with mock.patch.object(assessment, 'MAX_FINDINGS', 2):
            findings, summary = assessment.assess(data)
        self.assertEqual(len(findings), 3)
        self.assertEqual(findings[-1]['rule'], 'workload.coverage')
        self.assertEqual(findings[-1]['status'], 'unknown')
        self.assertTrue(summary['assessment_truncated'])


if __name__ == '__main__':
    unittest.main()
