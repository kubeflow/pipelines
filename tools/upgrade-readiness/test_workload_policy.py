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
"""Target authorization predictions preserve exact server request semantics."""

import copy
import json
import unittest
from unittest import mock

from target_rbac import evaluate_access
import workload_policy as policy


def bundle():
    return dict(
        policy_contract=policy.CONTRACT,
        target_revision='a' * 40,
        multi_user=True,
        shared_read=False,
        compiler_patch_empty=True,
        plugins_disabled=True,
        rbac_complete=True,
        rbac_only=True,
        service_account_mode='enforce',
        workflow_identity_mode='enforce',
        default_service_account='pipeline-runner',
        allowed_service_accounts=['custom', 'helper'],
        rbac=[],
        identities=[
            dict(resource_kind='run', resource_id='run-id', user='caller')
        ])


def inventory():
    return dict(
        runs=[
            dict(
                run_id='run-id',
                namespace='team',
                experiment_id='exp',
                service_account='custom',
                pipeline_spec=dict(
                    apiVersion='argoproj.io/v1alpha1',
                    kind='Workflow',
                    spec=dict(
                        serviceAccountName='custom',
                        templates=[
                            dict(
                                name='main',
                                container=dict(
                                    image='busybox', args=['private-payload']))
                        ])))
        ],
        experiments=[dict(experiment_id='exp', namespace='team')],
        pipelines=[],
        pipeline_versions=[],
        recurring_runs=[])


def grant(target,
          resource='serviceaccounts',
          verbs=None,
          group='',
          names=None,
          user='caller',
          namespace='team'):
    rule = dict(apiGroups=[group], resources=[resource], verbs=verbs or ['use'])
    if names is not None:
        rule['resourceNames'] = names
    name = 'rule-' + str(len(target['rbac']))
    target['rbac'].extend([
        dict(
            kind='Role',
            metadata=dict(name=name, namespace=namespace),
            rules=[rule]),
        dict(
            kind='RoleBinding',
            metadata=dict(name=name, namespace=namespace),
            roleRef=dict(
                kind='Role', name=name, apiGroup='rbac.authorization.k8s.io'),
            subjects=[
                dict(
                    kind='User',
                    name=user,
                    apiGroup='rbac.authorization.k8s.io')
            ])
    ])


def results(source, target, rule):
    return [
        item for item in policy.assess(source, target) if item['rule'] == rule
    ]


def status(source, target, rule):
    return results(source, target, rule)[0]['status']


class WorkloadPolicyTest(unittest.TestCase):

    def test_validation_requires_explicit_versioned_settings(self):
        target = bundle()
        self.assertIs(policy.validate(target), target)
        for key, value in [('policy_contract', 'other'),
                           ('target_revision', True), ('multi_user', 1),
                           ('workflow_identity_mode', 'off'),
                           ('allowed_service_accounts', [False]),
                           ('allowed_service_accounts', ['x' * 1025]),
                           ('allowed_service_accounts', ['\x1ccustom']),
                           ('plugins_disabled', 'yes')]:
            with self.subTest(key=key), self.assertRaises(ValueError):
                policy.validate(dict(target, **{key: value}))
        with self.assertRaises(ValueError):
            policy.validate(dict(target, identities=target['identities'] * 2))
        with self.assertRaises(ValueError):
            policy.validate(dict(target, rbac=[{}] * 10001))
        with self.assertRaises(ValueError):
            policy.validate(
                dict(
                    target,
                    access_checks=[
                        dict(
                            operation='shell', namespace='team', user='caller')
                    ]))

    def test_default_is_exempt_but_custom_account_needs_exact_identity(self):
        source, target = inventory(), bundle()
        target['identities'] = []
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'unknown')
        source['runs'][0]['service_account'] = 'pipeline-runner'
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'no_issue_detected')
        self.assertEqual(
            status(source, target, 'workload.createRun'), 'unknown')

    def test_named_grant_uses_no_inferred_service_account_groups(self):
        source, target = inventory(), bundle()
        grant(target, names=['custom'])
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'no_issue_detected')
        target['rbac'][1]['subjects'] = [
            dict(
                kind='Group',
                name='system:authenticated',
                apiGroup='rbac.authorization.k8s.io')
        ]
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'policy_rejection')
        target['rbac_complete'] = False
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'unknown')

    def test_audit_allowlist_denial_does_not_hide_unresolved_sar(self):
        source, target = inventory(), bundle()
        target.update(allowed_service_accounts=[], rbac_complete=False)
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'policy_rejection')
        target['service_account_mode'] = 'audit'
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'unknown')
        grant(target, names=['custom'])
        self.assertEqual(
            status(source, target, 'workload.mainAccount'),
            'operational_impact')

    def test_allowlist_trims_entries_without_expanding_literal_star(self):
        source, target = inventory(), bundle()
        grant(target, names=['custom'])
        target['allowed_service_accounts'] = [' custom ', '', '  ']
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'no_issue_detected')
        for entries in (['*'], [''], ['   '],
                        ['{{workflow.parameters.account}}']):
            with self.subTest(entries=entries):
                target['allowed_service_accounts'] = entries
                self.assertEqual(
                    status(source, target, 'workload.mainAccount'),
                    'policy_rejection')
        target['multi_user'] = False
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'policy_rejection')

    def test_workflow_mode_is_independent_from_main_account_mode(self):
        source, target = inventory(), bundle()
        grant(target, names=['custom'])
        source['runs'][0]['pipeline_spec']['spec']['templates'][0][
            'serviceAccountName'] = 'helper'
        target['service_account_mode'] = 'audit'
        self.assertEqual(
            status(source, target, 'workload.embeddedAccount'),
            'policy_rejection')
        target['workflow_identity_mode'] = 'audit'
        self.assertEqual(
            status(source, target, 'workload.embeddedAccount'),
            'operational_impact')
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'no_issue_detected')

    def test_explicit_main_override_does_not_hide_same_account_in_template(
            self):
        source, target = inventory(), bundle()
        source['runs'][0]['service_account'] = 'pipeline-runner'
        source['runs'][0]['pipeline_spec']['spec']['templates'][0][
            'serviceAccountName'] = 'custom'
        self.assertEqual(
            status(source, target, 'workload.embeddedAccount'),
            'policy_rejection')
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'no_issue_detected')

    def test_template_defaults_executor_and_inline_accounts_are_inspected(self):
        source, target = inventory(), bundle()
        spec = source['runs'][0]['pipeline_spec']['spec']
        spec['executor'] = dict(serviceAccountName='executor')
        spec['templateDefaults'] = dict(serviceAccountName='helper')
        spec['templates'][0]['steps'] = [[
            dict(inline=dict(serviceAccountName='nested'))
        ]]
        findings = results(source, target, 'workload.embeddedAccount')
        self.assertEqual(len(findings), 3)
        self.assertTrue(
            all(item['status'] == 'policy_rejection' for item in findings))

    def test_unresolved_workflow_inputs_never_establish_identity_coverage(self):
        for change in ('compiler', 'plugins', 'v2', 'external', 'patch',
                       'stored', 'gc', 'malformed'):
            with self.subTest(change=change):
                source, target = inventory(), bundle()
                workflow = source['runs'][0]['pipeline_spec']
                if change == 'compiler':
                    target['compiler_patch_empty'] = False
                elif change == 'plugins':
                    target.pop('plugins_disabled')
                elif change == 'v2':
                    source['runs'][0]['pipeline_spec'] = dict(
                        pipelineInfo=dict(name='pipeline'), root={})
                elif change == 'external':
                    workflow['spec']['workflowTemplateRef'] = dict(
                        name='external')
                elif change == 'patch':
                    workflow['spec'][
                        'podSpecPatch'] = '{"serviceAccountName":"private-payload"}'
                elif change == 'stored':
                    workflow['status'] = dict(storedWorkflowSpec={})
                elif change == 'gc':
                    workflow['spec']['artifactGC'] = dict(
                        strategy='OnWorkflowCompletion')
                else:
                    workflow['spec']['dag'] = 'private-payload'
                self.assertEqual(
                    status(source, target, 'workload.identityCoverage'),
                    'unknown')
                self.assertNotIn('private-payload',
                                 json.dumps(policy.assess(source, target)))

    def test_single_user_skips_sar_but_preserves_account_allowlist(self):
        source, target = inventory(), bundle()
        target.update(multi_user=False, identities=[])
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'no_issue_detected')
        target['allowed_service_accounts'] = []
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'policy_rejection')

    def test_conflicting_experiment_namespace_cannot_pass(self):
        source, target = inventory(), bundle()
        source['experiments'][0]['namespace'] = 'other'
        self.assertEqual(
            [item['status'] for item in policy.assess(source, target)],
            ['unknown'])

    def test_missing_experiment_cannot_establish_namespace_authorization(self):
        source, target = inventory(), bundle()
        source['experiments'] = []
        source['runs'][0]['_readiness_namespace_evidence'] = 'unresolved_parent'
        grant(target, names=['custom'])
        self.assertEqual(
            [item['status'] for item in policy.assess(source, target)],
            ['unknown'])

    def test_invalid_account_is_not_treated_as_omitted(self):
        for value in (False, [], {}):
            with self.subTest(value=value):
                source, target = inventory(), bundle()
                source['runs'][0]['service_account'] = value
                self.assertEqual(
                    status(source, target, 'workload.mainAccount'), 'unknown')

    def test_embedded_configured_default_is_exempt_in_both_identity_modes(self):
        source, target = inventory(), bundle()
        grant(target, names=['custom'])
        source['runs'][0]['pipeline_spec']['spec']['templates'][0][
            'serviceAccountName'] = 'pipeline-runner'
        for mode in ('enforce', 'audit'):
            with self.subTest(mode=mode):
                target['workflow_identity_mode'] = mode
                self.assertEqual(
                    status(source, target, 'workload.embeddedAccount'),
                    'no_issue_detected')

    def test_latest_snapshot_cannot_supply_future_default_or_identity_coverage(
            self):
        source, target = inventory(), bundle()
        spec = source['runs'][0].pop('pipeline_spec')
        source['runs'][0]['service_account'] = ''
        source['runs'][0]['pipeline_version_reference'] = dict(
            pipeline_id='pipeline')
        source['runs'][0]['_readiness_version_reference'] = dict(
            kind='moving_latest',
            resolution='observed',
            pipeline_version_id='version')
        source['pipelines'] = [
            dict(
                pipeline_id='pipeline', name='pipeline-name', namespace='team')
        ]
        source['pipeline_versions'] = [
            dict(
                pipeline_version_id='version',
                pipeline_id='pipeline',
                pipeline_spec=spec)
        ]
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'unknown')
        self.assertEqual(
            status(source, target, 'workload.identityCoverage'), 'unknown')
        source['runs'][0]['service_account'] = 'pipeline-runner'
        self.assertEqual(
            status(source, target, 'workload.mainAccount'), 'no_issue_detected')

    def test_lifecycle_hooks_prevent_exhaustive_identity_coverage(self):
        hooks = {
            'exit': {
                'templateRef': {
                    'name': 'external',
                    'template': 'cleanup'
                }
            }
        }
        for location in ('workflow', 'dag', 'steps'):
            source, target = inventory(), bundle()
            target['plugins_disabled'] = True
            workflow = {
                'serviceAccountName': 'pipeline-runner',
                'templates': []
            }
            source['runs'][0]['pipeline_spec'] = {
                'kind': 'Workflow',
                'apiVersion': 'argoproj.io/v1alpha1',
                'spec': workflow
            }
            if location == 'workflow':
                workflow['hooks'] = hooks
            elif location == 'dag':
                workflow['templates'] = [{'dag': {'tasks': [{'hooks': hooks}]}}]
            else:
                workflow['templates'] = [{'steps': [[{'hooks': hooks}]]}]
            with self.subTest(location=location):
                self.assertEqual(
                    status(source, target, 'workload.identityCoverage'),
                    'unknown')

    def test_resource_templates_are_outside_inline_identity_coverage(self):
        source, target = inventory(), bundle()
        target['plugins_disabled'] = True
        source['runs'][0]['pipeline_spec'] = {
            'kind': 'Workflow',
            'spec': {
                'templates': [{
                    'resource': {
                        'action':
                            'create',
                        'manifest':
                            'apiVersion: v1\nkind: Pod\nspec:\n  serviceAccountName: helper\n'
                    }
                }]
            }
        }
        self.assertEqual(
            status(source, target, 'workload.identityCoverage'), 'unknown')

    def test_empty_external_reference_objects_are_not_absent(self):
        for spec in ({
                'workflowTemplateRef': {}
        }, {
                'templates': [{
                    'dag': {
                        'tasks': [{
                            'templateRef': {}
                        }]
                    }
                }]
        }):
            source, target = inventory(), bundle()
            target['plugins_disabled'] = True
            source['runs'][0]['pipeline_spec'] = {
                'kind': 'Workflow',
                'spec': spec
            }
            self.assertEqual(
                status(source, target, 'workload.identityCoverage'), 'unknown')

    def test_conflicting_collected_pipeline_source_cannot_grant(self):
        source, target = inventory(), bundle()
        source['runs'][0]['_readiness_collection_errors'] = [
            'conflicting_pipeline_source'
        ]
        findings = policy.assess(source, target)
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0]['rule'], 'workload.authorization')
        self.assertEqual(findings[0]['status'], 'unknown')

    def test_inspection_budget_counts_nodes_tasks_and_step_groups(self):
        for templates in ([{} for _ in range(10)
                          ], [dict(dag=dict(tasks=[{} for _ in range(10)]))
                             ], [dict(steps=[[] for _ in range(10)])],
                          [dict(steps=[[{} for _ in range(10)]])]):
            with self.subTest(templates=templates):
                source, target = inventory(), bundle()
                source['runs'][0]['pipeline_spec']['spec'][
                    'templates'] = templates
                before = copy.deepcopy((source, target))
                with mock.patch.object(policy, 'MAX_INSPECTION_ITEMS', 8):
                    self.assertEqual(
                        status(source, target, 'workload.identityCoverage'),
                        'unknown')
                self.assertEqual((source, target), before)

    def test_inline_inspection_boundary_preserves_partial_accounts(self):
        workflow = dict(
            spec=dict(
                serviceAccountName='custom',
                templates=[
                    dict(
                        steps=[[dict(inline=dict(
                            serviceAccountName='helper'))]])
                ]))
        before = copy.deepcopy(workflow)
        with mock.patch.object(policy, 'MAX_INSPECTION_ITEMS', 4):
            self.assertEqual(
                policy._workflow_accounts(workflow, 'custom'),
                ({'custom'}, False))
        with mock.patch.object(policy, 'MAX_INSPECTION_ITEMS', 5):
            self.assertEqual(
                policy._workflow_accounts(workflow, 'custom'),
                ({'custom', 'helper'}, True))
        self.assertEqual(workflow, before)

    def test_flattened_steps_do_not_modify_dag_task_input(self):
        workflow = dict(
            spec=dict(templates=[
                dict(
                    dag=dict(tasks=[
                        dict(inline=dict(serviceAccountName='dag-helper'))
                    ]),
                    steps=[[
                        dict(inline=dict(serviceAccountName='step-helper'))
                    ]])
            ]))
        before = copy.deepcopy(workflow)
        with mock.patch.object(policy, 'MAX_INSPECTION_ITEMS', 7):
            self.assertEqual(
                policy._workflow_accounts(workflow, 'custom'),
                ({'dag-helper', 'step-helper'}, True))
        self.assertEqual(workflow, before)

    def test_finding_budget_retains_first_results_and_stops_resource_work(self):
        source, target = inventory(), bundle()
        first = policy.assess(source, target)
        for index in range(20):
            source['runs'].append(
                dict(
                    copy.deepcopy(source['runs'][0]),
                    run_id='later-' + str(index)))
        before = copy.deepcopy((source, target))
        with mock.patch.object(policy, 'MAX_FINDINGS', 6), mock.patch.object(
                policy, '_source',
                wraps=policy._source) as resolve, mock.patch.object(
                    policy,
                    '_workflow_accounts',
                    wraps=policy._workflow_accounts) as inspect:
            findings = policy.assess(source, target)
        self.assertEqual(len(findings), 6)
        self.assertEqual(findings[:len(first)], first)
        self.assertEqual(findings[-2]['rule'], 'workload.mainAccount')
        self.assertEqual(findings[-1]['rule'], 'workload.coverage')
        self.assertEqual(findings[-1]['status'], 'unknown')
        self.assertIn('remaining workload and access checks are unassessed',
                      findings[-1]['evidence'])
        self.assertEqual(resolve.call_count, 2)
        self.assertEqual(inspect.call_count, 1)
        self.assertEqual((source, target), before)

    def test_finding_budget_stops_between_embedded_account_decisions(self):
        source, target = inventory(), bundle()
        source['runs'][0]['pipeline_spec']['spec']['templates'] = [
            dict(serviceAccountName=account)
            for account in ('alpha', 'beta', 'gamma')
        ]
        with mock.patch.object(policy, 'MAX_FINDINGS', 4), mock.patch.object(
                policy, '_account_decision',
                wraps=policy._account_decision) as decide:
            findings = policy.assess(source, target)
        self.assertEqual([item['rule'] for item in findings], [
            'workload.mainAccount', 'workload.embeddedAccount',
            'workload.embeddedAccount', 'workload.coverage'
        ])
        self.assertEqual(decide.call_count, 3)
        self.assertNotIn('gamma', json.dumps(findings))

    def test_finding_budget_is_shared_with_access_checks(self):
        target = bundle()
        target['access_checks'] = [
            dict(operation='manage_viewers', namespace='team', user='caller')
            for _ in range(20)
        ]
        with mock.patch.object(policy, 'MAX_FINDINGS', 5), mock.patch.object(
                policy, '_access', wraps=policy._access) as access:
            findings = policy.assess({}, target)
        self.assertEqual(len(findings), 5)
        self.assertTrue(
            all(item['rule'] == 'access.manage_viewers'
                for item in findings[:-1]))
        self.assertEqual(findings[-1]['rule'], 'workload.coverage')
        self.assertEqual(access.call_count, 4)
        with mock.patch.object(policy, 'MAX_FINDINGS', 6):
            combined = policy.assess(inventory(), target)
        self.assertEqual([item['rule'] for item in combined], [
            'workload.mainAccount', 'workload.identityCoverage',
            'workload.createRun', 'workload.pipelineRead',
            'access.manage_viewers', 'workload.coverage'
        ])
        with mock.patch.object(policy, 'MAX_FINDINGS', 2):
            self.assertEqual(policy.assess({}, bundle()), [])
        with mock.patch.object(policy, 'MAX_FINDINGS', 5):
            self.assertNotIn(
                'workload.coverage',
                [item['rule'] for item in policy.assess(inventory(), bundle())])

    def test_schedule_requires_explicit_controller_and_unnamed_create_grant(
            self):
        source, target = inventory(), bundle()
        run = source['runs'].pop()
        run['recurring_run_id'] = run.pop('run_id')
        source['recurring_runs'] = [run]
        grant(
            target,
            resource='runs',
            verbs=['create'],
            group=policy.PIPELINES_GROUP,
            names=['run-id'])
        self.assertEqual(
            status(source, target, 'workload.createRun'), 'unknown')
        target['identities'][0]['resource_kind'] = 'recurring_run'
        self.assertEqual(
            status(source, target, 'workload.createRun'), 'policy_rejection')
        target['rbac'][0]['rules'][0].pop('resourceNames')
        self.assertEqual(
            status(source, target, 'workload.createRun'), 'no_issue_detected')

    def test_pinned_pipeline_ownership_and_private_namespace_boundary(self):
        source, target = inventory(), bundle()
        source['runs'][0].pop('pipeline_spec')
        source['runs'][0]['pipeline_version_reference'] = dict(
            pipeline_id='pipeline', pipeline_version_id='version')
        source['pipelines'] = [
            dict(
                pipeline_id='pipeline', name='pipeline-name', namespace='other')
        ]
        source['pipeline_versions'] = [
            dict(
                pipeline_version_id='version',
                pipeline_id='pipeline',
                pipeline_spec=dict(pipelineInfo=dict(name='pipeline'), root={}))
        ]
        grant(
            target,
            resource='pipelines',
            verbs=['get'],
            group=policy.PIPELINES_GROUP,
            namespace='other')
        self.assertEqual(
            status(source, target, 'workload.pipelineRead'), 'policy_rejection')
        target['shared_read'] = True
        self.assertEqual(
            status(source, target, 'workload.pipelineRead'),
            'no_issue_detected')
        source['pipeline_versions'][0]['pipeline_id'] = 'other-owner'
        self.assertEqual(
            status(source, target, 'workload.pipelineRead'), 'unknown')

    def test_access_operations_match_server_verbs_resources_and_names(self):
        target = bundle()
        target['access_checks'] = [
            dict(
                operation='read_logs',
                namespace='team',
                user='caller',
                resource_name='workflow-name'),
            dict(
                operation='manage_viewers',
                namespace='team',
                user='caller',
                resource_name='ignored'),
            dict(
                operation='upload_version',
                namespace='team',
                user='caller',
                resource_name='version-name')
        ]
        grant(
            target,
            resource='runs',
            verbs=['readLog'],
            group=policy.PIPELINES_GROUP,
            names=['workflow-name'])
        grant(
            target,
            resource='viewers',
            verbs=['get', 'create', 'delete'],
            group='kubeflow.org',
            names=['ignored'])
        grant(
            target,
            resource='pipelines',
            verbs=['create'],
            group=policy.PIPELINES_GROUP,
            names=['version-name'])
        self.assertEqual(
            status({}, target, 'access.read_logs'), 'no_issue_detected')
        self.assertTrue(
            all(item['status'] == 'policy_rejection'
                for item in results({}, target, 'access.manage_viewers')))
        self.assertEqual(
            status({}, target, 'access.upload_version'), 'policy_rejection')

    def test_shared_read_does_not_bypass_run_log_authorization(self):
        target = bundle()
        target['shared_read'] = True
        target['access_checks'] = [
            dict(
                operation=operation,
                namespace='team',
                user='caller',
                resource_name='workflow')
            for operation in ('read_logs', 'viewer_logs')
        ]
        self.assertEqual(
            status({}, target, 'access.read_logs'), 'policy_rejection')
        self.assertEqual(
            status({}, target, 'access.viewer_logs'), 'no_issue_detected')

    def test_upload_uses_known_empty_name_and_rejects_named_grant(self):
        target = bundle()
        target['access_checks'] = [
            dict(operation='upload_pipeline', namespace='team', user='caller')
        ]
        grant(
            target,
            resource='pipelines',
            verbs=['create'],
            group=policy.PIPELINES_GROUP,
            names=['permitted'])
        self.assertEqual(
            status({}, target, 'access.upload_pipeline'), 'policy_rejection')
        target['rbac'][0]['rules'][0].pop('resourceNames')
        self.assertEqual(
            status({}, target, 'access.upload_pipeline'), 'no_issue_detected')

    def test_field_aliases_and_findings_are_sanitized(self):
        source, target = inventory(), bundle()
        run = source['runs'][0]
        for snake, camel in (('run_id', 'runId'), ('experiment_id',
                                                   'experimentId'),
                             ('pipeline_spec', 'pipelineSpec'),
                             ('service_account', 'serviceAccount')):
            run[camel] = run.pop(snake)
        report = policy.assess(source, target)
        self.assertNotIn('private-payload', json.dumps(report))
        self.assertNotIn('caller', json.dumps(report))
        self.assertEqual(
            set(report[0]), {
                'rule', 'status', 'resource', 'evidence', 'action',
                'verification'
            })

    def test_generic_rbac_subresource_matching_and_literal_resource_names(self):
        target = bundle()
        grant(target, resource='*/log', verbs=['get'], names=['pod'])
        args = (target['rbac'], 'caller', [], 'team', 'get', '', 'pods/log')
        self.assertEqual(
            evaluate_access(*args, resource_name='pod', complete=True),
            'allowed')
        self.assertEqual(
            evaluate_access(*args, resource_name='other', complete=True),
            'denied')
        target['rbac'][0]['rules'][0]['resourceNames'] = ['*']
        self.assertEqual(
            evaluate_access(*args, resource_name='pod', complete=True),
            'denied')


if __name__ == '__main__':
    unittest.main()
