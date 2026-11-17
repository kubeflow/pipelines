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
"""Test admission decisions and conflict receipts against changing queue
snapshots."""

import json
import subprocess
import unittest
from pathlib import Path

MODULE = Path(__file__).with_name('queue_dependency_conflicts.js')


def exercise(steps):
    script = r"""
const {checkDependencyConflict} = require(process.argv[1]);
const fs = require('node:fs');
const steps = JSON.parse(fs.readFileSync(0, 'utf8'));
let statuses = [], cache = new Map();
async function step(options) {
  if (!options.reuseCache) cache = new Map();
  const a = 'a'.repeat(40), b = 'b'.repeat(40), c = 'c'.repeat(40), d = 'd'.repeat(40);
  const base = options.base || a, head = options.head || b, queued = options.queued || c;
  const context = {repo: {owner:'org',repo:'repo'}, runId:99};
  const pr = {number:1, head:{sha:head}, base:{ref:'main',sha:base}};
  const entries = options.empty ? [] : [{id:options.entryId || 'first-entry',pullRequest:{number:2,headRefOid:queued},
    baseCommit:{oid:base},headCommit:options.unbuilt ? null : {oid:options.candidate || d}}];
  if (options.bypassed) entries.unshift({pullRequest:{number:3,headRefOid:'9'.repeat(40)},baseCommit:{oid:base},headCommit:null});
  if (options.append) entries.push({pullRequest:{number:4,headRefOid:'8'.repeat(40)},baseCommit:{oid:d},headCommit:null});
  if (options.reverse) entries.reverse();
  const calls = [], logs = [];
  let time = 0;
  const manifest = 'frontend/package.json';
  const github = {rest:{repos:{
    compareCommitsWithBasehead:async ({basehead}) => {
      calls.push(basehead);
      if (options.readError) throw Error('unavailable');
      const paths = basehead.endsWith(head) ? options.incoming : options.files;
      return {data:{files:(paths || [manifest]).map(p => typeof p === 'string' ? {filename:p} : p)}};
    },
    createCommitStatus:async receipt => {
      calls.push('receipt');
      statuses = [{...receipt, creator:{type:'Bot',login:'github-actions[bot]'}}];
      if (options.badWriter) statuses[0].creator.login = 'different[bot]';
    },
  }, pulls: {
    listFiles:async ({page}) => {
      calls.push('page'+page);
      return {data:(options.pages?.[page-1] || []).map(filename => ({filename}))};
    },
    get:async () => ({data:{head:{sha:options.racedHead || head},base:{ref:'main'},changed_files:options.fileCount}}),
  }}};
  const defer = await checkDependencyConflict({github, context, pr, entries, base,
    core:{info:msg => logs.push(msg)}, cache, statuses, now:() => time,
    probe:async () => {
      calls.push('merge-tree');
      time += options.duration || 0;
      if (options.probeError) throw Error('fetch failed');
      return options.clean ? null : {paths:options.conflicts || [manifest]};
    }});
  return {defer,calls,logs};
}
(async () => {
  const out = [];
  for (const options of steps) out.push(await step(options));
  console.log(JSON.stringify(out));
})();
"""
    result = subprocess.run(
        ['node', '-e', script, str(MODULE)],
        input=json.dumps(steps),
        capture_output=True,
        text=True,
        check=True,
    )
    return json.loads(result.stdout)


class DependencyConflictTest(unittest.TestCase):
    def test_only_defer_actual_conflicts_on_shared_dependency_paths(self):
        steps = [
            {'empty': True},
            {'incoming': ['frontend/src/test.js']},
            {'files': ['frontend/server/package.json']},
            {'clean': True},
            {'conflicts': ['frontend/src/test.js']},
            {'probeError': True},
            {'readError': True},
            {
                'incoming': [
                    {
                        'filename': 'frontend/new-name',
                        'previous_filename': 'frontend/package.json',
                    }
                ]
            },
        ]
        results = exercise(steps)
        self.assertEqual([r['defer'] for r in results], [False] * 7 + [True])
        self.assertEqual(results[0]['calls'], [])
        for res in results[:3]:
            self.assertNotIn('merge-tree', res['calls'])
        self.assertIn('receipt', results[-1]['calls'])

    def test_receipt_is_bound_to_pr_and_ordered_queue_inputs(self):
        for changed in [
            {'head': 'e' * 40},
            {'base': 'e' * 40},
            {'queued': 'e' * 40},
            {'candidate': 'e' * 40},
            {'entryId': 'replacement-entry'},
            {'unbuilt': True},
        ]:
            with self.subTest(changed=changed):
                first, unchanged, updated = exercise(
                    [{}, {}, dict(changed, clean=True)]
                )
                self.assertTrue(first['defer'])
                self.assertTrue(unchanged['defer'])
                self.assertFalse(updated['defer'])
                self.assertEqual(unchanged['calls'], [])
                self.assertIn('merge-tree', updated['calls'])

    def test_receipt_invalidates_on_queue_membership_and_order(self):
        original, extended, reordered = exercise(
            [{}, {'append': True}, {'append': True, 'reverse': True}]
        )
        for result in [original, extended, reordered]:
            self.assertTrue(result['defer'])
            self.assertIn('merge-tree', result['calls'])

    def test_skips_bypassed_unbuilt_member_while_including_unbuilt_tail(self):
        bypassed, trailing = exercise(
            [{'bypassed': True, 'files': []}, {'unbuilt': True}]
        )
        self.assertFalse(bypassed['defer'])
        self.assertNotIn('a' * 40 + '...' + '9' * 40, bypassed['calls'])
        self.assertTrue(trailing['defer'])

    def test_untrusted_receipt_does_not_defer_pr(self):
        _, second = exercise([{'badWriter': True}, {'clean': True}])
        self.assertFalse(second['defer'])
        self.assertIn('merge-tree', second['calls'])

    def test_path_discovery_is_reused_within_sweep(self):
        first, second = exercise(
            [
                {'clean': True},
                {'reuseCache': True, 'head': 'e' * 40, 'clean': True},
            ]
        )
        self.assertEqual(len(first['calls']), 3)
        self.assertEqual(len(second['calls']), 2)
        self.assertTrue(second['calls'][0].endswith('e' * 40))

    def test_large_diff_uses_paginated_files_and_validates_head(self):
        large = {
            'incoming': [f'code/{i}' for i in range(300)],
            'pages': [
                [f'code/{i}' for i in range(100)],
                ['frontend/package.json'],
            ],
            'fileCount': 101,
        }
        yes, raced = exercise(
            [dict(large, badWriter=True), dict(large, racedHead='e' * 40)]
        )
        self.assertTrue(yes['defer'])
        self.assertIn('page2', yes['calls'])
        self.assertFalse(raced['defer'])
        self.assertNotIn('merge-tree', raced['calls'])

    def test_sweep_git_budget_charges_time_spent_including_failed_probes(self):
        results = exercise(
            [
                {'duration': 25000, 'probeError': True},
                {'reuseCache': True, 'duration': 1000, 'clean': True},
                {'reuseCache': True, 'duration': 20000, 'clean': True},
                {'reuseCache': True},
            ]
        )
        self.assertIn('merge-tree', results[2]['calls'])
        self.assertFalse(results[3]['defer'])
        self.assertNotIn('merge-tree', results[3]['calls'])
