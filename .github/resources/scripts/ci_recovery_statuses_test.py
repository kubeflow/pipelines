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
"""Exercise bounded status discovery against complete and partial responses."""

import json
import subprocess
import unittest
from pathlib import Path

MODULE = Path(__file__).with_name('ci_recovery_statuses.js')


def exercise(options=None):
    script = r"""
const {readRecoveryStatuses} = require(process.argv[1]);
const options = JSON.parse(process.argv[2]);
const heads = options.heads || Array.from({length: options.count ?? 1}, (_, i) =>
  (i + 1).toString(16).padStart(40, '0'));
const calls = [];
let active = 0, maxActive = 0;
const github = {graphql: async (query, variables) => {
  calls.push({query, variables});
  active++;
  maxActive = Math.max(active, maxActive);
  await Promise.resolve();
  active--;
  if (options.error === 'throw') throw Error('GraphQL partial data');
  const repository = {nameWithOwner: 'UPSTREAM/REPO'};
  for (const [name, head] of Object.entries(variables)) {
    if (!name.startsWith('sha')) continue;
    repository[`head${name.slice(3)}`] = {__typename: 'Commit', oid: head,
      status: {context: {context: 'ci-passed', state: options.state || 'SUCCESS',
        description: options.nullDescription ? null : 'base-policy-stamp'}}};
  }
  const commit = repository.head0;
  switch (options.error) {
    case 'repository': repository.nameWithOwner = 'fork/repo'; break;
    case 'missing': delete repository.head0; break;
    case 'null-commit': repository.head0 = null; break;
    case 'type': commit.__typename = 'Tree'; break;
    case 'oid': commit.oid = 'f'.repeat(40); break;
    case 'missing-status': delete commit.status; break;
    case 'missing-context': commit.status = {}; break;
    case 'context': commit.status.context.context = 'another-check'; break;
    case 'state': commit.status.context.state = 'GREEN'; break;
    case 'description': delete commit.status.context.description; break;
  }
  if (options.absent === 'status') commit.status = null;
  if (options.absent === 'context') commit.status.context = null;
  return {repository, ...(options.error === 'partial' ? {errors: [{message: 'partial'}]} : {})};
}};
readRecoveryStatuses({github, owner: 'upstream', repo: 'repo', heads}).then(statuses => {
  console.log(JSON.stringify({statuses: [...statuses], calls, maxActive}));
}).catch(error => console.log(JSON.stringify({error: error.message, calls, maxActive})));
"""
    result = subprocess.run(
        ['node', '-e', script, str(MODULE), json.dumps(options or {})],
        capture_output=True,
        text=True,
        check=True,
    )
    return json.loads(result.stdout)


class RecoveryStatusesTest(unittest.TestCase):
    def test_recovery_discovery_batches_101_heads(self):
        script = r"""
const {recoveryCandidates} = require(process.argv[1]);
const prs = Array.from({length: 101}, (_, i) => ({number: i + 1,
  head: {sha: (i + 1).toString(16).padStart(40, '0')}, labels: [],
  base: {ref: 'master', sha: 'b'.repeat(40)}}));
const calls = {rest: 0, graphql: 0};
const github = {rest: {pulls: {list: {}}, repos: {getCombinedStatusForRef: {}}},
  paginate: async () => {calls.rest++; return prs;},
  graphql: async (query, variables) => {
    calls.graphql++;
    const repository = {nameWithOwner: 'o/r'};
    for (const [name, oid] of Object.entries(variables)) {
      if (name.startsWith('sha')) repository[`head${name.slice(3)}`] = {
        __typename: 'Commit', oid, status: {context: {
          context: 'ci-passed', state: 'FAILURE', description: 'failed'}}};
    }
    return {repository};
  }};
github.paginate.iterator = async function* () {
  calls.rest++;
  yield {data: {statuses: [{context: 'ci-passed', state: 'failure', description: 'failed'}]}};
};
recoveryCandidates({github, context: {repo: {owner: 'o', repo: 'r'}}})
  .then(result => console.log(JSON.stringify({calls, count: result.length})))
  .catch(error => {console.error(error); process.exit(1);});
"""
        result = subprocess.run(
            ['node', '-e', script, str(MODULE.with_name('ci_passed.js'))],
            capture_output=True,
            text=True,
            check=True,
        )
        actual = json.loads(result.stdout)
        self.assertEqual(actual['count'], 101)
        self.assertEqual(actual['calls'], {'rest': 1, 'graphql': 3})

    def test_batches_exact_heads_in_upstream_repository(self):
        result = exercise({'count': 101})
        self.assertNotIn('error', result)
        self.assertEqual(len(result['statuses']), 101)
        self.assertEqual(len(result['calls']), 3)
        self.assertEqual(result['maxActive'], 1)
        self.assertEqual(
            [len(call['variables']) - 2 for call in result['calls']],
            [50, 50, 1],
        )
        for call in result['calls']:
            self.assertEqual(call['variables']['owner'], 'upstream')
            self.assertEqual(call['variables']['repo'], 'repo')
            self.assertIn('object(oid: $sha0)', call['query'])
            self.assertIn('context(name: "ci-passed")', call['query'])
            self.assertNotIn('history', call['query'])

    def test_normalizes_all_states_and_preserves_description(self):
        for state in ['ERROR', 'EXPECTED', 'FAILURE', 'PENDING', 'SUCCESS']:
            with self.subTest(state=state):
                result = exercise({'state': state})
                self.assertEqual(
                    result['statuses'][0][1],
                    {
                        'state': state.lower(),
                        'description': 'base-policy-stamp',
                    },
                )
        result = exercise({'nullDescription': True})
        self.assertIsNone(result['statuses'][0][1]['description'])

    def test_absent_status_is_distinct_from_incomplete_response(self):
        for absent in ['status', 'context']:
            with self.subTest(absent=absent):
                result = exercise({'absent': absent})
                self.assertIsNone(result['statuses'][0][1])

    def test_rejects_partial_or_malformed_evidence(self):
        for error in [
            'throw',
            'partial',
            'repository',
            'missing',
            'null-commit',
            'type',
            'oid',
            'missing-status',
            'missing-context',
            'context',
            'state',
            'description',
        ]:
            with self.subTest(error=error):
                result = exercise({'error': error, 'count': 51})
                self.assertIn('error', result)
                self.assertNotIn('statuses', result)
                self.assertEqual(len(result['calls']), 1)

    def test_deduplicates_heads_without_requests_for_empty_input(self):
        head = 'a' * 40
        result = exercise({'heads': [head, head]})
        self.assertEqual(len(result['statuses']), 1)
        self.assertEqual(len(result['calls']), 1)
        self.assertEqual(len(result['calls'][0]['variables']), 3)
        result = exercise({'heads': []})
        self.assertEqual(result['calls'], [])
        self.assertEqual(result['statuses'], [])

    def test_rejects_nonexact_heads_before_network_access(self):
        for head in ['master', 'abc123', '', None, 123, 'a' * 39]:
            with self.subTest(head=head):
                result = exercise({'heads': [head]})
                self.assertIn('error', result)
                self.assertEqual(result['calls'], [])


if __name__ == '__main__':
    unittest.main()
