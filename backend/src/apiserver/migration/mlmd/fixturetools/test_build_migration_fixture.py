"""Check design branches without changing the released source snapshot."""

import base64
import json
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import Mock, call

from build_migration_fixture import (
    build_expected,
    dispositions,
    identity,
    source_from_sql,
)
from google.protobuf.json_format import ParseDict
from google.protobuf.struct_pb2 import Struct
from run_fixture import capture_runs


class MigrationFixtureDesignTest(unittest.TestCase):
    def setUp(self) -> None:
        self.source = source_from_sql()

    def test_released_parameters_preserve_values_and_verified_producers(
        self,
    ) -> None:
        expected, unresolved = build_expected(self.source)
        tasks = {row['source']: row['fields'] for row in expected['tasks']}
        producer = tasks['mlmd.executions[id=2]']
        consumer = tasks['mlmd.executions[id=4]']
        self.assertEqual(
            [
                {
                    'parameterKey': 'seed',
                    'value': 'fixture',
                    'type': 'COMPONENT_INPUT',
                }
            ],
            producer.get('InputParameters'),
        )
        self.assertEqual(
            [
                {
                    'parameterKey': 'Output',
                    'value': 'fixture-produced',
                    'type': 'OUTPUT',
                    'producer': {'taskName': 'produce'},
                }
            ],
            producer.get('OutputParameters'),
        )
        self.assertEqual(
            [
                {
                    'parameterKey': 'value',
                    'value': 'fixture-produced',
                    'type': 'TASK_OUTPUT_INPUT',
                    'producer': {'taskName': 'produce'},
                }
            ],
            consumer.get('InputParameters'),
        )
        self.assertEqual(
            [
                {
                    'parameterKey': 'pipelinechannel--loop-item-param-2',
                    'value': ['one', 'two'],
                }
            ],
            tasks['mlmd.executions[id=23]'].get('InputParameters'),
        )
        self.assertEqual(
            [
                {
                    'parameterKey': 'pipelinechannel--loop-item-param-2',
                    'value': 'one',
                    'type': 'ITERATOR_INPUT',
                    'producer': {'taskName': 'for-loop-3', 'iteration': '0'},
                }
            ],
            tasks['mlmd.executions[id=25]'].get('InputParameters'),
        )
        self.assertTrue(
            any(
                item['source'] == 'mlmd.executions[id=23]'
                and 'execution_properties.inputs' in item['fields']
                for item in unresolved
            )
        )
        self.assertTrue(
            any(
                item['status'] == 'mapped'
                and item['targets']
                == [
                    f'expected.tasks[UUID={consumer["UUID"]}].fields.InputParameters'
                ]
                for item in dispositions(self.source, expected)
            )
        )

    def test_unverified_input_retains_the_captured_value_and_type(self) -> None:
        value = {'list': ['zero', 0, False, None], 'empty': {}}
        encoded = base64.b64encode(
            ParseDict({'value': value}, Struct()).SerializeToString()
        ).decode('ascii')
        source_parameter = next(
            row
            for row in self.source['mlmd']['execution_properties']
            if row['execution_id'] == 4 and row['name'] == 'inputs'
        )
        source_parameter['string_value'] = f'mlmd-struct::{encoded}'
        expected, _ = build_expected(self.source)
        consumer = next(
            row['fields']
            for row in expected['tasks']
            if row['source'] == 'mlmd.executions[id=4]'
        )
        (entry,) = consumer['InputParameters']
        self.assertEqual({'parameterKey': 'value', 'value': value}, entry)
        self.assertIs(type(entry['value']['list'][1]), float)
        self.assertIs(entry['value']['list'][2], False)
        self.assertTrue(
            any(
                item['status'] == 'mapped_pending_design'
                and item['targets']
                == [
                    f'expected.tasks[UUID={consumer["UUID"]}].fields.InputParameters'
                ]
                for item in dispositions(self.source, expected)
            )
        )

    def test_native_fields_preserve_pods_iterations_cache_links_and_time(
        self,
    ) -> None:
        expected, _ = build_expected(self.source)
        tasks = {row['target_uuid']: row['fields'] for row in expected['tasks']}
        cache_row = next(
            row for row in self.source['kfp']['tasks'] if row['MLMDExecutionID']
        )
        task = tasks[identity('execution', int(cache_row['MLMDExecutionID']))]
        self.assertEqual(cache_row['StartedTimestamp'], task['StartedInSec'])
        self.assertEqual(cache_row['FinishedTimestamp'], task['FinishedInSec'])
        self.assertTrue(task['Pods'][0]['name'])
        self.assertTrue(task['Pods'][0]['uid'])
        self.assertTrue(
            any(
                t.get('TypeAttrs', {}).get('iterationIndex') == '0'
                for t in tasks.values()
            )
        )
        self.assertTrue(
            any(t.get('CachedTaskUUID') in tasks for t in tasks.values())
        )
        source_artifact = self.source['mlmd']['artifacts'][0]
        target = next(
            row['fields']
            for row in expected['artifacts']
            if row['target_uuid'] == identity('artifact', source_artifact['id'])
        )
        self.assertEqual(
            source_artifact['create_time_since_epoch'] // 1000,
            target['CreatedAtInSec'],
        )
        self.assertEqual(
            source_artifact['last_update_time_since_epoch'] // 1000,
            target['LastUpdateInSec'],
        )

    def test_joined_cache_fields_point_to_the_native_execution_task(
        self,
    ) -> None:
        expected, _ = build_expected(self.source)
        by_field = {
            item['source']: item for item in dispositions(self.source, expected)
        }
        mapped_fields = {
            'RunUUID': 'RunUUID',
            'Namespace': 'Namespace',
            'Fingerprint': 'StatusMetadata.customProperties.legacy_cache_fingerprint',
            'CreatedTimestamp': 'CreatedAtInSec',
            'StartedTimestamp': 'StartedInSec',
            'FinishedTimestamp': 'FinishedInSec',
        }
        cache_rows = [
            row for row in self.source['kfp']['tasks'] if row['MLMDExecutionID']
        ]
        self.assertEqual(12, len(cache_rows))
        for row in cache_rows:
            row_path = f'kfp.tasks[{row["UUID"]}]'
            source_ref = f'kfp.tasks[UUID={row["UUID"]}]'
            target_id = identity('execution', int(row['MLMDExecutionID']))
            targets = {
                field: f'expected.old_task_aliases[source={source_ref}].target_uuid'
                for field in ('UUID', 'MLMDExecutionID')
            }
            targets.update(
                {
                    field: f'expected.tasks[UUID={target_id}].fields.{target_field}'
                    for field, target_field in mapped_fields.items()
                }
            )
            for field, target in targets.items():
                with self.subTest(row=row_path, field=field):
                    item = by_field[f'{row_path}.{field}']
                    self.assertEqual('mapped', item['status'])
                    self.assertEqual([target], item['targets'])

    def test_linked_cache_pod_name_wins_over_a_different_mlmd_pod(self) -> None:
        cache_row = next(
            row for row in self.source['kfp']['tasks'] if row['MLMDExecutionID']
        )
        cache_row['PodName'] = 'legacy-reported-pod'
        payload = json.loads(cache_row['Payload'])
        payload['PodName'] = cache_row['PodName']
        cache_row['Payload'] = json.dumps(payload)

        expected, _ = build_expected(self.source)
        execution_id = int(cache_row['MLMDExecutionID'])
        task = next(
            row['fields']
            for row in expected['tasks']
            if row['target_uuid'] == identity('execution', execution_id)
        )
        self.assertEqual([{'name': 'legacy-reported-pod'}], task['Pods'])

    def test_workflow_reports_are_skipped_and_saved_manifests_are_preserved(
        self,
    ) -> None:
        expected, _ = build_expected(self.source)
        reports = {
            f'kfp.tasks[UUID={row["UUID"]}]'
            for row in self.source['kfp']['tasks']
            if not row['MLMDExecutionID']
        }
        self.assertEqual(179, len(reports))
        self.assertEqual(
            reports,
            {
                row['source']
                for row in expected['skips']
                if row['source'].startswith('kfp.tasks[')
            },
        )
        self.assertEqual(
            {
                f'mlmd.executions[id={row["id"]}]'
                for row in self.source['mlmd']['executions']
            },
            {row['source'] for row in expected['tasks']},
        )
        self.assertEqual(
            {
                row['UUID']: row['WorkflowRuntimeManifest']
                for row in self.source['kfp']['run_details']
            },
            {
                row['fields']['UUID']: row['fields']['WorkflowRuntimeManifest']
                for row in expected['run_details']
            },
        )

    def test_orphaned_run_context_is_skipped(self) -> None:
        types = {row['id']: row['name'] for row in self.source['mlmd']['types']}
        context = next(
            row
            for row in self.source['mlmd']['contexts']
            if types[row['type_id']] == 'system.PipelineRun'
        )
        context['name'] = 'missing-run'
        expected, _ = build_expected(self.source)
        skipped = {row['source']: row for row in expected['skips']}
        self.assertIn(f'mlmd.contexts[id={context["id"]}]', skipped)
        associated = [
            row['execution_id']
            for row in self.source['mlmd']['associations']
            if row['context_id'] == context['id']
        ]
        for execution_id in associated:
            self.assertIn(f'mlmd.executions[id={execution_id}]', skipped)
        self.assertTrue(
            all(
                row['severity'] == 'warning'
                for row in skipped.values()
                if row['source'].startswith('mlmd.')
            )
        )
        fields = dispositions(self.source, expected)
        for table in (
            'contexts',
            'context_properties',
            'associations',
            'attributions',
            'parent_contexts',
            'executions',
        ):
            self.assertTrue(
                any(
                    row['status'] == 'skip'
                    and row['source'].startswith(f'mlmd.{table}[')
                    for row in fields
                ),
                table,
            )

    def test_missing_parent_keeps_child_and_event(self) -> None:
        parent = next(
            row
            for row in self.source['mlmd']['execution_properties']
            if row['name'] == 'parent_dag_id' and row['execution_id'] == 2
        )
        parent['int_value'] = 999999
        expected, _ = build_expected(self.source)
        child = next(
            row
            for row in expected['tasks']
            if row['source'] == 'mlmd.executions[id=2]'
        )
        self.assertIsNone(child['fields']['ParentTaskUUID'])
        self.assertTrue(
            any(
                'parent execution:999999 is missing or skipped'
                in entry['reason']
                for entry in expected['diagnostics']
            )
        )
        self.assertTrue(
            any(
                link['fields']['TaskID'] == child['target_uuid']
                for link in expected['artifact_tasks']
            )
        )

    def test_cyclic_parents_detach_only_the_cycle(self) -> None:
        for edges, cyclic in (
            ({2: 2, 4: 2}, {2}),
            ({2: 4, 4: 2}, {2, 4}),
        ):
            with self.subTest(edges=edges):
                source = source_from_sql()
                for prop in source['mlmd']['execution_properties']:
                    if (
                        prop['name'] == 'parent_dag_id'
                        and prop['execution_id'] in edges
                    ):
                        prop['int_value'] = edges[prop['execution_id']]
                expected, _ = build_expected(source)
                tasks = {
                    row['target_uuid']: row['fields']
                    for row in expected['tasks']
                }
                for child, parent in edges.items():
                    task_id = identity('execution', child)
                    self.assertEqual(
                        None
                        if child in cyclic
                        else identity('execution', parent),
                        tasks[task_id]['ParentTaskUUID'],
                    )
                    if child in cyclic:
                        self.assertIn(
                            {
                                'source': f'mlmd.executions[id={child}]',
                                'reason': f'parent execution:{parent} forms a cycle',
                                'severity': 'warning',
                            },
                            expected['diagnostics'],
                        )
                self.assertTrue(
                    any(
                        link['fields']['TaskID'] == identity('execution', 2)
                        for link in expected['artifact_tasks']
                    )
                )

    def test_cross_run_parent_is_rejected(self) -> None:
        parent_link = next(
            row
            for row in self.source['mlmd']['execution_properties']
            if row['execution_id'] == 2 and row['name'] == 'parent_dag_id'
        )
        parent_link['int_value'] = 5  # Root execution of a different run.
        expected, _ = build_expected(self.source)
        child = next(
            row
            for row in expected['tasks']
            if row['source'] == 'mlmd.executions[id=2]'
        )
        self.assertIsNone(child['fields']['ParentTaskUUID'])
        self.assertTrue(
            any(
                row['source'] == child['source']
                and 'different run' in row['reason']
                for row in expected['diagnostics']
            )
        )

    def test_unresolved_event_type_and_paths_have_pending_dispositions(
        self,
    ) -> None:
        events = self.source['mlmd']['events']
        paths = self.source['mlmd']['event_paths']
        events[0]['type'] = 3
        composite_id = events[0]['id']
        indexed_id = events[1]['id']
        composite = next(
            row for row in paths if row['event_id'] == composite_id
        )
        indexed = next(row for row in paths if row['event_id'] == indexed_id)
        index_step = {'is_index_step': 1, 'step_index': 0, 'step_key': ''}
        paths.append({**composite, **index_step})
        indexed.update(index_step)

        expected, _ = build_expected(self.source)
        skipped = {row['source'] for row in expected['skips']}
        for event_id in (composite_id, indexed_id):
            self.assertIn(f'mlmd.events[id={event_id}]', skipped)
        fields = {
            row['source']: row for row in dispositions(self.source, expected)
        }
        self.assertEqual(
            'skip', fields[f'mlmd.events[{composite_id}].type']['status']
        )
        self.assertEqual(
            'skip',
            fields[f'mlmd.event_paths[{paths.index(indexed)}].step_index'][
                'status'
            ],
        )

    def test_integer_metric_and_metadata_scalar_are_preserved(self) -> None:
        types = {row['id']: row['name'] for row in self.source['mlmd']['types']}
        metric = next(
            row
            for row in self.source['mlmd']['artifacts']
            if types[row['type_id']] == 'system.Metrics'
        )
        accuracy = next(
            row
            for row in self.source['mlmd']['artifact_properties']
            if row['artifact_id'] == metric['id'] and row['name'] == 'accuracy'
        )
        accuracy['double_value'] = None
        accuracy['int_value'] = 1
        expected, _ = build_expected(self.source)
        targets = [
            row['fields']
            for row in expected['artifacts']
            if row['source'] == f'mlmd.artifacts[id={metric["id"]}]'
        ]
        self.assertEqual({'accuracy', 'loss'}, {row['Name'] for row in targets})
        self.assertEqual(
            1,
            next(
                row['NumberValue']
                for row in targets
                if row['Name'] == 'accuracy'
            ),
        )
        self.assertTrue(
            all(
                isinstance(row['Metadata']['store_session_info'], str)
                for row in targets
            )
        )
        self.assertTrue(
            any(
                row['source'].endswith('.int_value')
                and f'metric=accuracy' in ''.join(row['targets'])
                and row['status'] == 'mapped'
                for row in dispositions(self.source, expected)
            )
        )

    def test_empty_metric_artifact_and_its_links_have_skip_diagnostics(
        self,
    ) -> None:
        property_rows = self.source['mlmd']['artifact_properties']
        artifact_id = next(
            row['artifact_id']
            for row in property_rows
            if row['name'] == 'accuracy'
        )
        property_rows[:] = [
            row
            for row in property_rows
            if row['artifact_id'] != artifact_id
            or row['name'] not in ('accuracy', 'loss')
        ]
        expected, _ = build_expected(self.source)
        skipped = {row['source']: row for row in expected['skips']}
        self.assertEqual(
            'metrics artifact has no scalar numeric property',
            skipped[f'mlmd.artifacts[id={artifact_id}]']['reason'],
        )
        events = [
            row
            for row in self.source['mlmd']['events']
            if row['artifact_id'] == artifact_id
        ]
        self.assertTrue(events)
        for event in events:
            self.assertEqual(
                'warning', skipped[f'mlmd.events[id={event["id"]}]']['severity']
            )

    def test_unknown_execution_type_is_warning_skip(self) -> None:
        importer_type = next(
            row
            for row in self.source['mlmd']['types']
            if row['name'] == 'system.ImporterExecution'
        )
        importer_type['name'] = 'custom.UnknownExecution'
        expected, _ = build_expected(self.source)
        self.assertTrue(
            any(
                row['reason'] == 'unknown execution type'
                and row['severity'] == 'warning'
                for row in expected['skips']
            )
        )

    def test_missing_optional_property_gets_warning_and_default(self) -> None:
        properties = self.source['mlmd']['execution_properties']
        row = next(row for row in properties if row['name'] == 'display_name')
        properties.remove(row)
        expected, _ = build_expected(self.source)
        source_ref = f'mlmd.executions[id={row["execution_id"]}]'
        self.assertTrue(
            any(
                item['source'] == source_ref and item['severity'] == 'warning'
                for item in expected['diagnostics']
            )
        )
        task = next(
            item for item in expected['tasks'] if item['source'] == source_ref
        )
        self.assertEqual('', task['fields']['DisplayName'])


class CaptureRunsTest(unittest.TestCase):
    def test_wait_failure_records_the_run_and_resumes_before_submitting_cache(
        self,
    ) -> None:
        cases = {'success-first', 'success-cache'}
        for failure in (TimeoutError('wait expired'), KeyboardInterrupt()):
            with (
                self.subTest(failure=type(failure)),
                tempfile.TemporaryDirectory() as directory,
            ):
                output = Path(directory)
                run_record = output / 'runs.json'
                client = Mock()
                client.create_run_from_pipeline_package.side_effect = [
                    SimpleNamespace(run_id='producer-id'),
                    SimpleNamespace(run_id='cache-id'),
                ]

                def interrupted_wait(run_id: str, *, timeout: int) -> None:
                    self.assertEqual('producer-id', run_id)
                    self.assertEqual(900, timeout)
                    self.assertEqual(
                        [
                            {
                                'case': 'success-first',
                                'run_id': run_id,
                                'status': 'SUBMITTED',
                            }
                        ],
                        json.loads(run_record.read_text()),
                    )
                    raise failure

                client.wait_for_run_completion.side_effect = interrupted_wait
                with self.assertRaises(type(failure)):
                    capture_runs(client, output, cases)
                self.assertEqual(
                    1, client.create_run_from_pipeline_package.call_count
                )

                client.wait_for_run_completion.side_effect = None
                client.wait_for_run_completion.return_value = SimpleNamespace(
                    state='SUCCEEDED'
                )
                capture_runs(client, output, cases)
                self.assertEqual(
                    ['success-first', 'success-cache'],
                    [
                        c.kwargs['run_name']
                        for c in client.create_run_from_pipeline_package.call_args_list
                    ],
                )
                self.assertEqual(
                    [
                        call('producer-id', timeout=900),
                        call('producer-id', timeout=900),
                        call('cache-id', timeout=900),
                    ],
                    client.wait_for_run_completion.call_args_list,
                )
                self.assertEqual(
                    [
                        {
                            'case': 'success-first',
                            'run_id': 'producer-id',
                            'status': 'SUCCEEDED',
                        },
                        {
                            'case': 'success-cache',
                            'run_id': 'cache-id',
                            'status': 'SUCCEEDED',
                        },
                    ],
                    json.loads(run_record.read_text()),
                )

    def test_cache_only_capture_requires_a_successful_producer(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory)
            (output / 'runs.json').write_text(
                json.dumps(
                    [
                        {
                            'case': 'success-first',
                            'run_id': 'producer-id',
                            'status': 'FAILED',
                        },
                    ]
                )
            )
            client = Mock()
            with self.assertRaisesRegex(
                RuntimeError, 'success-first must succeed'
            ):
                capture_runs(client, output, {'success-cache'})
            client.create_run_from_pipeline_package.assert_not_called()


if __name__ == '__main__':
    unittest.main()
