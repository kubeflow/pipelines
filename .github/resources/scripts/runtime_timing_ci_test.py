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
"""Guard opt-in timing setup, preserved environment and bounded capture."""

import copy
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import collect_runtime_server_timings as collect
import configure_runtime_timings as configure
import yaml

ROOT = Path(__file__).resolve().parents[3]


class RuntimeTimingCITest(unittest.TestCase):

    def test_real_capture_pipeline_filters_unselected_server_lines(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            fixture = root / 'fixture'
            fixture.mkdir()
            correlation = '10000000-0000-4000-8000-000000000001'
            client = {
                'correlation_id': correlation,
                'method': '/api.RunService/CreateTask',
                'total_ms': 10,
                'status': 'OK'
            }
            (fixture / 'pod.log').write_text('KFP_RUNTIME_CLIENT_TIMING ' +
                                             json.dumps(client))
            (root / 'runtime-timing-configuration.json').write_text(
                json.dumps({
                    'enabled': True,
                    'capture_since': '2026-10-12T00:00:00+00:00'
                }))
            server = {
                'correlation_id': correlation,
                'method': 'CreateTask',
                'total_ms': 9,
                'phases_ms': {
                    'token_review': 5
                }
            }
            executable = root / 'kubectl'
            executable.write_text(
                '#!/usr/bin/env python3\nprint("unrelated private diagnostic")\nprint('
                + repr('KFP_RUNTIME_SERVER_TIMING ' + json.dumps(server)) +
                ')\n')
            executable.chmod(0o755)
            env = dict(
                os.environ,
                PROFILE_RUNTIME_TIMINGS='true',
                KFP_E2E_PROFILE_DIR=str(root),
                PROFILE_NAMESPACE='kubeflow',
                PATH=str(root) + os.pathsep + os.environ['PATH'])
            subprocess.run([os.sys.executable,
                            str(Path(collect.__file__))],
                           env=env,
                           check=True,
                           timeout=10)
            result = (root / 'server-timings.jsonl').read_text()
            self.assertEqual(json.loads(result), server)
            self.assertNotIn('private', result)
            summary = json.loads(
                (root / 'server-timing-collection.json').read_text())
            self.assertEqual(summary['filter_exit_code'], 0)
            self.assertEqual(summary['kubectl_exit_code'], 0)

    def test_server_profile_is_part_of_initial_manifest_without_rollout(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root /
             'kustomization.yaml').write_text('resources: [deployment.yaml]')
            deployment = {
                'apiVersion': 'apps/v1',
                'kind': 'Deployment',
                'metadata': {
                    'name': 'ml-pipeline'
                },
                'spec': {
                    'template': {
                        'spec': {
                            'containers': [{
                                'name':
                                    'ml-pipeline-api-server',
                                'image':
                                    'same',
                                'env': [{
                                    'name': 'EXISTING',
                                    'value': 'preserved'
                                }]
                            }]
                        }
                    }
                }
            }
            (root / 'deployment.yaml').write_text(yaml.safe_dump(deployment))
            command = [
                str(ROOT /
                    '.github/resources/scripts/render-profiled-manifests.sh'),
                str(root)
            ]
            for enabled, tuned in (('false', 'false'), ('true', 'false'),
                                   ('false', 'true'), ('true', 'true')):
                result = subprocess.run(
                    command,
                    env=dict(
                        os.environ,
                        PROFILE_RUNTIME_TIMINGS=enabled,
                        TUNE_KUBERNETES_CLIENT=tuned),
                    text=True,
                    capture_output=True,
                    check=True,
                    timeout=20)
                actual = yaml.safe_load(
                    result.stdout)['spec']['template']['spec']['containers'][0]
                self.assertIn({
                    'name': 'EXISTING',
                    'value': 'preserved'
                }, actual['env'])
                self.assertEqual(
                    any(e['name'] == configure.NAME for e in actual['env']),
                    enabled == 'true')
                environment = {e['name']: e.get('value') for e in actual['env']}
                self.assertEqual(
                    environment.get('CLIENTQPS'),
                    '20' if tuned == 'true' else None)
                self.assertEqual(
                    environment.get('CLIENTBURST'),
                    '40' if tuned == 'true' else None)
                self.assertEqual(actual['image'], 'same')
            self.assertEqual(
                yaml.safe_load((root / 'deployment.yaml').read_text()),
                deployment)
        configure_source = Path(configure.__file__).read_text()
        self.assertNotIn("'rollout'", configure_source)
        self.assertNotIn("'set', 'env'", configure_source)
        deploy = yaml.safe_load(
            (ROOT / '.github/actions/deploy/action.yml').read_text())
        self.assertEqual(deploy['inputs']['profile_runtime_timings']['default'],
                         'false')
        self.assertEqual(deploy['inputs']['tune_kubernetes_client']['default'],
                         'false')
        steps = deploy['runs']['steps']
        self.assertLess(
            next(i for i, step in enumerate(steps)
                 if step.get('id') == 'deploy-kfp'),
            next(i for i, step in enumerate(steps)
                 if step.get('id') == 'forward-api-port'))
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/e2e-test.yml').read_text())
        callers = [
            step for job in workflow['jobs'].values()
            for step in job.get('steps', [])
            if step.get('uses') == './.github/actions/deploy'
        ]
        self.assertEqual(len(callers), 4)
        self.assertTrue(
            all(step['with']['tune_kubernetes_client'] == 'true'
                for step in callers))
        api_workflow = yaml.safe_load(
            (ROOT / '.github/workflows/api-server-tests.yml').read_text())
        api_callers = [
            step for job in api_workflow['jobs'].values()
            for step in job.get('steps', [])
            if step.get('uses') == './.github/actions/deploy'
        ]
        self.assertEqual(len(api_callers), 3)
        self.assertTrue(
            all(step['with']['tune_kubernetes_client'] == 'true'
                for step in api_callers))
        self.assertTrue(
            all(step['with']['profile_runtime_timings'] ==
                "${{ github.event_name != 'workflow_dispatch' }}"
                for step in callers))

    def test_preserves_defaults_and_all_unrelated_environment(self):
        patch = {
            'containers': [{
                'name': 'other',
                'env': [{
                    'name': 'OTHER',
                    'value': 'value'
                }]
            }, {
                'name': 'main',
                'env': [{
                    'name': 'FIXTURE',
                    'value': 'keep'
                }, {
                    'name': 'SECRET_REF',
                    'valueFrom': {
                        'secretKeyRef': {
                            'name': 's',
                            'key': 'k'
                        }
                    }
                }],
                'resources': {
                    'requests': {
                        'cpu': '100m'
                    }
                }
            }],
            'nodeSelector': {
                'node': 'same'
            },
            'initContainers': [{
                'name': 'init',
                'image': 'same'
            }]
        }
        defaults = {
            'spec': {
                'podSpecPatch': yaml.safe_dump(patch),
                'ttlStrategy': {
                    'secondsAfterCompletion': 3600
                },
                'templateDefaults': {
                    'retryStrategy': {
                        'limit': '2'
                    }
                }
            }
        }
        original = copy.deepcopy(defaults)
        result = configure.workflow_patch(
            {'data': {
                'workflowDefaults': yaml.safe_dump(defaults)
            }})
        actual = yaml.safe_load(result['data']['workflowDefaults'])
        updated = json.loads(actual['spec'].pop('podSpecPatch'))
        original['spec'].pop('podSpecPatch')
        self.assertEqual(actual, original)
        self.assertEqual(updated['containers'][1]['env'][-1], {
            'name': configure.NAME,
            'value': 'true'
        })
        updated['containers'][1]['env'].pop()
        self.assertEqual(updated, patch)

    def test_missing_patch_added_and_repeat_is_idempotent(self):
        first = configure.workflow_patch({'data': {}})
        second = configure.workflow_patch(first)
        self.assertEqual(first, second)
        defaults = yaml.safe_load(first['data']['workflowDefaults'])
        self.assertEqual(
            json.loads(defaults['spec']['podSpecPatch']), {
                'containers': [{
                    'name': 'main',
                    'env': [{
                        'name': configure.NAME,
                        'value': 'true'
                    }]
                }]
            })

    def test_invalid_patch_fails_without_mutating_cluster(self):
        for value in ('[]', 'containers: invalid',
                      'containers: [{name: main}, {name: main}]',
                      'containers: [{name: main, env: invalid}]'):
            with self.subTest(value=value):
                config = {
                    'data': {
                        'workflowDefaults':
                            yaml.safe_dump({'spec': {
                                'podSpecPatch': value
                            }})
                    }
                }
                with self.assertRaises(ValueError):
                    configure.workflow_patch(config)

    def test_default_disabled_performs_no_commands_or_reads(self):
        with mock.patch.dict(
                os.environ, {}, clear=True), mock.patch.object(
                    subprocess,
                    'run') as run, mock.patch.object(subprocess,
                                                     'Popen') as popen:
            configure.main()
            collect.main()
            run.assert_not_called()
            popen.assert_not_called()

    def test_collection_is_once_bounded_stream_and_records_overhead(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / 'runtime-timing-configuration.json').write_text(
                json.dumps({
                    'enabled': True,
                    'capture_since': '2026-10-12T12:00:00+00:00'
                }))
            logs = mock.Mock()
            filtered = mock.Mock()
            logs.wait.return_value = filtered.wait.return_value = 0
            logs.poll.return_value = filtered.poll.return_value = 0
            with mock.patch.dict(
                    os.environ, {
                        'PROFILE_RUNTIME_TIMINGS': 'true',
                        'KFP_E2E_PROFILE_DIR': str(root),
                        'PROFILE_NAMESPACE': 'kubeflow'
                    }), mock.patch.object(
                        subprocess, 'Popen', side_effect=[logs,
                                                          filtered]) as popen:
                collect.main()
            self.assertEqual(popen.call_count, 2)
            command = popen.call_args_list[0].args[0]
            self.assertIn('--limit-bytes=16777216', command)
            self.assertIn('--request-timeout=20s', command)
            self.assertIn('--since-time=2026-10-12T12:00:00+00:00', command)
            self.assertIs(popen.call_args_list[1].kwargs['stdin'], logs.stdout)
            summary = json.loads(
                (root / 'server-timing-collection.json').read_text())
            self.assertEqual(summary['filter_exit_code'], 0)
            self.assertGreater(summary['elapsed_ms'], 0)
            self.assertEqual(
                set(p.name for p in root.iterdir()), {
                    'runtime-timing-configuration.json',
                    'server-timing-collection.json'
                })

    def test_collection_timeout_kills_both_processes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / 'runtime-timing-configuration.json').write_text(
                json.dumps({
                    'enabled': True,
                    'capture_since': '2026-10-12T12:00:00+00:00'
                }))
            logs, filtered = mock.Mock(), mock.Mock()
            logs.poll.return_value = filtered.poll.return_value = None
            filtered.wait.side_effect = [
                subprocess.TimeoutExpired('filter', 30), 0
            ]
            with mock.patch.dict(
                    os.environ, {
                        'PROFILE_RUNTIME_TIMINGS': 'true',
                        'KFP_E2E_PROFILE_DIR': str(root),
                        'PROFILE_NAMESPACE': 'kubeflow'
                    }), mock.patch.object(
                        subprocess, 'Popen', side_effect=[logs, filtered]):
                with self.assertRaises(RuntimeError):
                    collect.main()
            logs.kill.assert_called_once()
            filtered.kill.assert_called_once()
            self.assertTrue(
                json.loads(
                    (root /
                     'server-timing-collection.json').read_text())['timed_out'])

    def test_action_order_keeps_collection_before_upload_and_is_opt_in(self):
        action = yaml.safe_load(
            (ROOT / '.github/actions/test-and-report/action.yml').read_text())
        steps = action['runs']['steps']
        names = [step['name'] for step in steps]
        self.assertLess(
            names.index('Enable runtime RPC timing for profiling'),
            names.index('Run Tests'))
        self.assertLess(
            names.index('Run Tests'),
            names.index('Collect structured runtime server timings'))
        self.assertLess(
            names.index('Collect structured runtime server timings'),
            names.index('Upload E2E lifecycle profiles'))
        for name in ('Enable runtime RPC timing for profiling',
                     'Collect structured runtime server timings'):
            step = steps[names.index(name)]
            self.assertIn("inputs.profile_e2e == 'true'", step['if'])
            self.assertTrue(step['continue-on-error'])
