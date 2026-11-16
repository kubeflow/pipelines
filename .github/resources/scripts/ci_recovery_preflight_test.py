#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
"""Keep scheduled recovery cheap without suppressing recoverable evidence."""

import json
import subprocess
import unittest
from pathlib import Path

MODULE = Path(__file__).with_name('ci_passed.js')
ROOT = Path(__file__).resolve().parents[3]


def exercise(options=None):
    script = r"""
const options = JSON.parse(process.argv[2]);
if(options.queueEnabled) require(require('node:path').join(require('node:path').dirname(process.argv[1]),
  'merge_queue_policy')).queueEnabled=async()=>true;
const {recoveryCandidates} = require(process.argv[1]);
const sha = 'b'.repeat(40), created = '2026-01-01T00:00:00Z';
const counts = {pulls:0, runs:0, base:0, arrival:0, merged:0, graphql:0};
const prs = Array.from({length:options.count || 1},(_,i)=>({number:i+1,state:'open',
 head:{sha:(i+1).toString(16).padStart(40,'0'),ref:'feature',repo:{full_name:'fork/r'}},
 base:{sha,ref:options.branch || 'master',repo:{full_name:'o/r'}},labels:options.labels || []}));
const content = `name: Required\non:\n  pull_request:\n    branches: [master, main, release-*]\n${options.filtered ? '    paths: [sdk/**]\n' : ''}jobs:\n  test:\n    runs-on: ubuntu-latest\n`;
const runsFor = head => {
 if(options.kind==='missing' || !options.kind)return [];
 const run={id:42,workflow_id:7,check_suite_id:9,run_attempt:1,
  event:'pull_request',path:'.github/workflows/required.yml',head_sha:head,
  head_branch:'feature',head_repository:{full_name:'fork/r'},repository:{full_name:'o/r'},
  created_at:created,run_started_at:created,status:'completed',conclusion:'success',
  ...options.run};
 if(options.kind==='fresh')run.created_at=run.run_started_at='2026-02-01T00:00:00Z';
 if(options.kind==='failed')run.conclusion='failure';
 if(options.kind==='active'){run.run_attempt=2;run.status='in_progress';run.conclusion=null;}
 return [run,...(options.newer ? [{...run,id:43,check_suite_id:10,created_at:'2026-02-01T00:00:00Z',
   run_started_at:'2026-02-01T00:00:00Z',conclusion:'success'}] : [])];
};
const listPulls={},associated={};
const github={rest:{pulls:{list:listPulls,get:async()=>{counts.merged++;return {data:{number:999,
 merged:true,merge_commit_sha:sha,merged_at:'2026-01-15T00:00:00Z',
 base:{ref:prs[0].base.ref,repo:{full_name:'o/r'}}}};}},
 repos:{listPullRequestsAssociatedWithCommit:associated},
 git:{getRef:async()=>{counts.base++;if(options.error==='base')throw Error('base unavailable');return {data:{ref:'refs/heads/'+(options.error==='ref-identity'?'other':prs[0].base.ref),object:{sha,type:'commit'}}};}},
 actions:{listWorkflowRunsForRepo:async({head_sha})=>{counts.runs++;if(options.error==='runs')throw Error('runs unavailable');
  const runs=runsFor(head_sha);return {data:{total_count:options.error==='truncated'?1000:runs.length,workflow_runs:runs}};}}},
 paginate:async(route)=>{if(route===listPulls){counts.pulls++;return prs;}
 if(route===associated){counts.arrival++;if(options.error==='arrival')throw Error('arrival unavailable');return [{number:999,merge_commit_sha:sha}];}
 throw Error('Unexpected REST pagination');},
 graphql:async(query,variables)=>{counts.graphql++;
 if(query.includes('RecoveryStatuses')){
  const repository={nameWithOwner:'o/r'};
  for(const [k,oid]of Object.entries(variables))if(k.startsWith('sha'))repository['head'+k.slice(3)]={__typename:'Commit',oid,
   status:options.absent?null:{context:{context:'ci-passed',state:options.status || 'FAILURE',description:'Previous assessment'}}};
  return {repository};
 }
 if(options.error==='inventory')throw Error('inventory unavailable');
 return {repository:{nameWithOwner:'o/r',object:{__typename:'Tree',entries:[{name:'required.yml',type:'blob',mode:33188,
  object:{__typename:'Blob',text:content,byteSize:Buffer.byteLength(content),isBinary:false,isTruncated:false}},
  ...(options.secondLane ? [{name:'another.yml',type:'blob',mode:33188,object:{__typename:'Blob',
    text:content,byteSize:Buffer.byteLength(content),isBinary:false,isTruncated:false}}]:[]),
  ...(options.filteredLane ? (()=>{const filtered=content.replace('jobs:', '    paths: [sdk/**]\njobs:');
    return [{name:'sdk.yml',type:'blob',mode:33188,object:{__typename:'Blob',text:filtered,
      byteSize:Buffer.byteLength(filtered),isBinary:false,isTruncated:false}}];})():[])]}}};
}};
recoveryCandidates({github,context:{repo:{owner:'o',repo:'r'},runId:99},root:process.argv[3]})
 .then(candidates=>console.log(JSON.stringify({candidates,counts})))
 .catch(error=>console.log(JSON.stringify({error:error.message,counts})));
"""
    result = subprocess.run(
        [
            'node',
            '-e',
            script,
            str(MODULE),
            json.dumps(options or {}),
            str(ROOT),
        ],
        check=True,
        capture_output=True,
        text=True,
    )
    return json.loads(result.stdout)


class RecoveryPreflightTest(unittest.TestCase):
    def test_121_dormant_failed_heads_do_not_spawn_full_assessments(self):
        result = exercise({'count': 121})
        self.assertNotIn('error', result)
        self.assertEqual(result['candidates'], [], result)
        self.assertEqual(result['counts']['runs'], 121)
        self.assertLessEqual(sum(result['counts'].values()), 129)

    def test_failure_with_missing_always_required_lane_is_deferred(self):
        self.assertEqual(exercise()['candidates'], [])

    def test_missing_pending_lane_still_reconciles_registration_deadline(self):
        self.assertEqual(len(exercise({'status': 'PENDING'})['candidates']), 1)

    def test_stale_pending_lane_keeps_filtered_registration_deadline(self):
        # sdk.yml is required for SDK changes but is absent from the run list.
        # A stale unconditional lane cannot suppress its registration deadline.
        result = exercise(
            {'kind': 'stale', 'status': 'PENDING', 'filteredLane': True}
        )
        self.assertEqual(len(result['candidates']), 1)
        self.assertEqual(result['counts']['runs'], 0)

    def test_stale_required_lane_keeps_existing_failure(self):
        self.assertEqual(exercise({'kind': 'stale'})['candidates'], [])

    def test_pending_heads_always_receive_full_reconciliation(self):
        for kind in ('missing', 'stale', 'failed', 'fresh', 'active'):
            with self.subTest(kind=kind):
                result = exercise({'kind': kind, 'status': 'PENDING'})
                self.assertEqual(len(result['candidates']), 1)
                self.assertEqual(result['counts']['runs'], 0)

    def test_failed_required_lane_does_not_spawn_full_assessment(self):
        self.assertEqual(exercise({'kind': 'failed'})['candidates'], [])

    def test_fresh_success_or_active_rerun_is_reconciled_immediately(self):
        for options in (
            {'kind': 'fresh'},
            {'kind': 'active'},
            {'kind': 'failed', 'newer': True},
            {'kind': 'fresh', 'run': {'run_attempt': 2}},
        ):
            with self.subTest(options=options):
                self.assertEqual(len(exercise(options)['candidates']), 1)

    def test_rerunning_older_execution_cannot_reuse_newer_execution_green(self):
        result = exercise(
            {
                'kind': 'failed',
                'newer': True,
                'run': {
                    'run_attempt': 2,
                    'run_started_at': '2026-03-01T00:00:00Z',
                },
            }
        )
        self.assertEqual(result['candidates'], [])

    def test_green_and_missing_status_never_use_negative_preflight(self):
        for options in ({'status': 'SUCCESS'}, {'absent': True}):
            with self.subTest(options=options):
                result = exercise(options)
                self.assertEqual(len(result['candidates']), 1)
                self.assertEqual(result['counts']['runs'], 0)

    def test_stale_green_label_gets_full_reconciliation(self):
        for status in ('FAILURE', 'PENDING'):
            with self.subTest(status=status):
                result = exercise(
                    {
                        'kind': 'stale',
                        'status': status,
                        'labels': [{'name': 'ci-passed'}],
                    }
                )
                self.assertEqual(len(result['candidates']), 1)
                self.assertEqual(result['counts']['runs'], 0)

    def test_path_filtered_lane_cannot_prove_missing_coverage(self):
        self.assertEqual(len(exercise({'filtered': True})['candidates']), 1)

    def test_release_base_never_applies_development_freshness(self):
        self.assertEqual(
            len(
                exercise({'kind': 'stale', 'branch': 'release-2.18'})[
                    'candidates'
                ]
            ),
            1,
        )

    def test_queue_admission_owns_base_freshness(self):
        result = exercise({'kind': 'stale', 'queueEnabled': True})
        self.assertEqual(len(result['candidates']), 1)
        self.assertEqual(result['counts']['arrival'], 0)

    def test_creation_at_arrival_is_still_stale(self):
        result = exercise(
            {'kind': 'stale', 'run': {'created_at': '2026-01-15T00:00:00Z'}}
        )
        self.assertEqual(result['candidates'], [])

    def test_missing_pending_lane_wins_over_other_stale_lane(self):
        result = exercise(
            {'kind': 'stale', 'status': 'PENDING', 'secondLane': True}
        )
        self.assertEqual(len(result['candidates']), 1)

    def test_late_successful_rerun_preserves_original_base_freshness(self):
        result = exercise(
            {
                'kind': 'stale',
                'run': {
                    'run_attempt': 2,
                    'run_started_at': '2026-02-01T00:00:00Z',
                },
            }
        )
        self.assertEqual(result['candidates'], [])

    def test_api_and_identity_ambiguity_falls_back_to_full_assessment(self):
        options = [
            {'error': error, 'kind': 'stale'}
            for error in (
                'base',
                'ref-identity',
                'runs',
                'inventory',
                'arrival',
                'truncated',
            )
        ] + [
            {'kind': 'stale', 'run': override}
            for override in (
                {'head_sha': 'f' * 40},
                {'head_branch': 'unrelated'},
                {'head_repository': {'full_name': 'other/r'}},
                {'repository': {'full_name': 'other/r'}},
                {'event': 'push'},
                {'run_attempt': 0},
                {'created_at': 'invalid'},
                {'status': 'mystery'},
            )
        ]
        for option in options:
            with self.subTest(option=option):
                result = exercise(option)
                self.assertNotIn('error', result)
                self.assertEqual(len(result['candidates']), 1)


if __name__ == '__main__':
    unittest.main()
