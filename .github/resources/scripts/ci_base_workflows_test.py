#!/usr/bin/env python3
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
"""Exercise trusted base workflow loading and the bounded upgrade opt-in."""

import copy
import json
from pathlib import Path
import subprocess
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/ci_expected_workflows.js'
GUARD = "vars.KFP_ENABLE_MLMD_UPGRADE_TESTS == 'true'"


def workflow(paths=None, jobs=None):
    return yaml.safe_dump(
        {
            'name': 'Example CI',
            'on': {
                'pull_request': {
                    'paths': paths or ['manifests/**']
                }
            },
            'jobs': jobs or {
                'test': {
                    'runs-on': 'ubuntu-latest'
                }
            },
        },
        sort_keys=False)


def tree(workflows):
    return {
        'repository': {
            'nameWithOwner': 'kubeflow/pipelines',
            'object': {
                '__typename':
                    'Tree',
                'entries': [{
                    'name': name,
                    'type': 'blob',
                    'mode': 33188,
                    'object': {
                        '__typename': 'Blob',
                        'text': content,
                        'byteSize': len(content.encode('utf-8')),
                        'isBinary': False,
                        'isTruncated': False
                    },
                } for name, content in workflows.items()]
            },
        }
    }


def exercise(workflows=None,
             response=None,
             enabled='',
             missing=None,
             skipped=None,
             base=None):
    if response is None:
        response = tree(workflows or {'frontend.yml': workflow()})
    fixture = {
        'response':
            response,
        'enabled':
            enabled,
        'missing':
            missing or [],
        'skipped':
            skipped or [],
        'base':
            base or {
                'sha': 'b' * 40,
                'ref': 'release-2.18',
                'repo': {
                    'full_name': 'kubeflow/pipelines'
                }
            },
    }
    script = r'''
const gate = require(process.argv[1]);
const fixture = JSON.parse(process.argv[2]);
const root = process.argv[3];
const requests = [];
const pr = {number: 7, changed_files: 1, base: fixture.base,
  head: {sha: 'malicious-head', ref: 'feature', repo: {full_name: 'attacker/pipelines'}}};
const github = {
  graphql: async (query, variables) => {requests.push({query, variables}); return fixture.response;},
  rest: {pulls: {listFiles: 'files'}, actions: {listWorkflowRunsForRepo: 'runs'}},
  paginate: async route => route === 'files' ? [{filename: 'manifests/ui.yaml'}] :
    fixture.response.repository.object.entries.filter(entry => !fixture.missing.includes(entry.name))
      .map(entry => ({id: 1, path: '.github/workflows/' + entry.name,
        event: 'pull_request', head_sha: pr.head.sha, head_branch: pr.head.ref,
        head_repository: pr.head.repo, status: 'completed',
        conclusion: fixture.skipped.includes(entry.name) ? 'skipped' : 'success',
        created_at: '2026-09-26T00:00:00Z', pull_requests: []})),
};
(async () => {
  try {
    const inventory = await gate.loadBaseInventory({github, owner: 'kubeflow', repo: 'pipelines',
      pullRequest: pr, root});
    const result = await gate.verifyExpectedWorkflows({github, owner: 'kubeflow', repo: 'pipelines',
      pullRequest: pr, ...inventory, enableMlmdUpgradeTests: fixture.enabled});
    console.log(JSON.stringify({...result, inventory: inventory.inventory, requests}));
  } catch (error) {console.log(JSON.stringify({error: error.message, requests}));}
})();
'''
    result = subprocess.run(
        ['node', '-e', script,
         str(MODULE),
         json.dumps(fixture),
         str(ROOT)],
        check=True,
        capture_output=True,
        text=True)
    return json.loads(result.stdout)


class BaseWorkflowsTest(unittest.TestCase):

    def test_release_triggers_and_release_only_lanes_replace_master_inventory(
            self):
        result = exercise(
            {
                'frontend.yml': workflow(),
                'ci-scripts-tests.yml': workflow(['.github/scripts/**']),
                'integration-tests-v1.yml': workflow(),
            },
            missing=['ci-scripts-tests.yml'])
        self.assertTrue(result['passed'], result)
        self.assertCountEqual(result['expected'], [
            '.github/workflows/frontend.yml',
            '.github/workflows/integration-tests-v1.yml',
        ])
        result = exercise({
            'ci-scripts-tests.yml': workflow(['manifests/**']),
        },
                          missing=['ci-scripts-tests.yml'])
        self.assertFalse(result['passed'])

    def test_missing_release_only_lane_blocks(self):
        result = exercise(
            {
                'frontend.yml': workflow(),
                'integration-tests-v1.yml': workflow(),
            },
            missing=['integration-tests-v1.yml'])
        self.assertFalse(result['passed'])
        self.assertIn('integration-tests-v1.yml', result['reasons'][0])

    def test_only_one_query_reads_base_repository_and_immutable_sha(self):
        result = exercise()
        self.assertTrue(result['passed'], result)
        self.assertEqual(len(result['requests']), 1)
        self.assertEqual(
            result['requests'][0]['variables'], {
                'owner': 'kubeflow',
                'repo': 'pipelines',
                'expression': 'b' * 40 + ':.github/workflows',
            })
        self.assertNotIn('malicious-head', result['requests'][0]['query'])
        for base in [
            {
                'sha': 'master',
                'repo': {
                    'full_name': 'kubeflow/pipelines'
                }
            },
            {
                'sha': 'b' * 40,
                'repo': {
                    'full_name': 'attacker/pipelines'
                }
            },
        ]:
            with self.subTest(base=base):
                result = exercise(base=base)
                self.assertIn('error', result)
                self.assertEqual(result['requests'], [])

    def test_incomplete_or_wrong_tree_fails_closed(self):
        good = tree({'frontend.yml': workflow()})
        variants = [{}, {'repository': None}]
        for change in [
                None, {}, {
                    '__typename': 'Blob'
                }, {
                    '__typename': 'Tree',
                    'entries': []
                }
        ]:
            candidate = copy.deepcopy(good)
            candidate['repository']['object'] = change
            variants.append(candidate)
        wrong_repo = copy.deepcopy(good)
        wrong_repo['repository']['nameWithOwner'] = 'attacker/pipelines'
        variants.append(wrong_repo)
        for response in variants:
            with self.subTest(response=response):
                self.assertIn('error', exercise(response=response))

    def test_invalid_duplicate_truncated_or_oversize_blob_fails_closed(self):
        mutations = [
            {
                'isBinary': True
            },
            {
                'isTruncated': True
            },
            {
                'text': None
            },
            {
                'byteSize': 0
            },
            {
                'byteSize': 1048577
            },
            {
                'byteSize': 1
            },
            {
                'isBinary': None
            },
            {
                'isTruncated': None
            },
        ]
        for change in mutations:
            response = tree({'frontend.yml': workflow()})
            response['repository']['object']['entries'][0]['object'].update(
                change)
            with self.subTest(change=change):
                self.assertIn('error', exercise(response=response))
        for change in [{
                'name': '../frontend.yml'
        }, {
                'mode': 40960
        }, {
                'type': 'tree'
        }, {
                'object': None
        }]:
            response = tree({'frontend.yml': workflow()})
            response['repository']['object']['entries'][0].update(change)
            with self.subTest(change=change):
                self.assertIn('error', exercise(response=response))
        response = tree({'frontend.yml': workflow()})
        entries = response['repository']['object']['entries']
        entries.append(copy.deepcopy(entries[0]))
        self.assertIn('error', exercise(response=response))

    def test_yaml_is_data_and_ambiguous_or_invalid_yaml_fails(self):
        for content in [
                "!!python/object/apply:builtins.print ['must not execute']",
                'name: First\nname: Second\non: pull_request\njobs: {}\n',
                'name: CI\non: [pull_request\n',
        ]:
            with self.subTest(content=content):
                self.assertIn('error', exercise({'frontend.yml': content}))

    def test_disabled_exact_whole_upgrade_workflow_is_disclosed(self):
        workflows = {
            'frontend.yml':
                workflow(),
            'upgrade-test.yml':
                workflow(
                    jobs={
                        'build': {
                            'if': '${{ ' + GUARD + ' }}'
                        },
                        'test': {
                            'if': '  ' + GUARD + '  '
                        },
                    }),
        }
        for enabled in ['', 'false', 'TRUE']:
            with self.subTest(enabled=enabled):
                result = exercise(
                    workflows, enabled=enabled, skipped=['upgrade-test.yml'])
                self.assertTrue(result['passed'], result)
                self.assertEqual(result['expected'],
                                 ['.github/workflows/frontend.yml'])
                self.assertEqual(result['disabled'][0]['path'],
                                 '.github/workflows/upgrade-test.yml')
        result = exercise(
            workflows, enabled='true', skipped=['upgrade-test.yml'])
        self.assertFalse(result['passed'])
        self.assertEqual(result['disabled'], [])
        self.assertTrue(exercise(workflows, enabled='true')['passed'])

    def test_unguarded_mixed_or_unrecognized_upgrade_jobs_remain_required(self):
        for jobs in [
            {
                'test': {
                    'runs-on': 'ubuntu-latest'
                }
            },
            {
                'build': {
                    'if': GUARD
                },
                'test': {
                    'runs-on': 'ubuntu-latest'
                }
            },
            {
                'test': {
                    'if': GUARD + ' || true'
                }
            },
            {
                'test': {
                    'if': False
                }
            },
        ]:
            with self.subTest(jobs=jobs):
                result = exercise(
                    {
                        'frontend.yml': workflow(),
                        'upgrade-test.yml': workflow(jobs=jobs)
                    },
                    skipped=['upgrade-test.yml'])
                self.assertFalse(result['passed'])
                self.assertEqual(result['disabled'], [])

    def test_same_guard_on_other_workflow_never_omits_requirement(self):
        result = exercise(
            {'frontend.yml': workflow(jobs={'test': {
                'if': GUARD
            }})},
            skipped=['frontend.yml'])
        self.assertFalse(result['passed'])
        self.assertEqual(result['disabled'], [])

    def test_workflow_installs_trusted_parser_and_invalidates_after_setup_failure(
            self):
        definition = yaml.safe_load(
            (ROOT / '.github/workflows/ci-checks.yml').read_text())
        job = definition['jobs']['check_ci_status']
        self.assertEqual(job['env']['CI_ENABLE_MLMD_UPGRADE_TESTS'],
                         '${{ vars.KFP_ENABLE_MLMD_UPGRADE_TESTS }}')
        steps = job['steps']
        prepare_index = next(
            i for i, step in enumerate(steps) if step.get('id') == 'prepare')
        before_prepare = steps[:prepare_index]
        self.assertTrue(
            any(
                step.get('uses') == 'actions/setup-python@v7'
                for step in before_prepare))
        self.assertTrue(
            any('.github/scripts/requirements.txt' in step.get('run', '')
                for step in before_prepare))
        self.assertEqual(steps[prepare_index]['if'], 'always() && !cancelled()')
        poll = next(step for step in steps if step.get('id') == 'poll')
        self.assertNotIn('always()', poll.get('if', ''))


if __name__ == '__main__':
    unittest.main()
