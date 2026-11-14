// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.


'use strict';

const path = require('node:path');
const {enabledBranches, queueEnabled} = require('./merge_queue_policy');
const {readQueue, inspectQueue, sweep} = require('./merge_queue_admission');
const {expectedWorkflows, readRef, queueRuns, matchingRuns, reconcile} = require('./ci_merge_group');
const {loadBaseInventory} = require('./ci_expected_workflows');
const QUEUE_REF = /^gh-readonly-queue\/(master|main)\/pr-[1-9][0-9]*-[A-Za-z0-9-]+$/;
const SHA = /^[0-9a-f]{40}$/;
const REGISTRATION_GRACE_MS = 120000;
const MAX_DISPATCH_ATTEMPTS = 3;

async function dispatchGroup({github, context, group, core, root,
  loadInventory = loadBaseInventory, now = () => Date.now()}) {
  const ref = QUEUE_REF.exec(group.branch || '');
  if (!ref || ref[1] !== group.baseRef || !SHA.test(group.head || '')) {
    throw new Error('Invalid queue dispatch candidate');
  }
  if (!await queueEnabled({github, context, branch: group.baseRef, root})) return;
  if (await readRef(github, context, group.branch) !== group.head) return;
  const base = await readRef(github, context, group.baseRef);
  const policy = await inspectQueue({github, context, group, root});
  if (policy === null) throw new Error('Queue policy disabled before dispatch validation.');
  const inventory = await loadInventory({github, ...context.repo, root,
    pullRequest: {base: {sha: base, ref: group.baseRef,
      repo: {full_name: `${context.repo.owner}/${context.repo.repo}`}}}});
  const expected = expectedWorkflows(inventory.inventory, inventory.workflowFiles, group.baseRef);
  const runs = await queueRuns(github, context, group);
  const statuses = await github.paginate(github.rest.repos.listCommitStatusesForRef, {
    ...context.repo, ref: group.head, per_page: 100,
  });
  if (!Array.isArray(statuses) || statuses.length >= 1000) {
    throw new Error('Incomplete queue dispatch reservations');
  }
  const current = async () => await readRef(github, context, group.branch) === group.head &&
    await readRef(github, context, group.baseRef) === base &&
    await inspectQueue({github, context, group, root}) === policy;
  const failures = [];
  for (const workflow of expected) {
    // Registered attempts, including failures/cancellations, are never silently
    // rerun. Latest evidence is handled by the publisher; deliberate retries use
    // Actions' rerun UI. The reservation handles only missing dispatch responses.
    if (matchingRuns(runs, context, group, workflow).length) continue;
    const statusContext = `ci-queue-dispatch/${path.basename(workflow)}`;
    const previous = statuses.find(status => status.context === statusContext);
    let attempt = 1;
    if (previous) {
      const receipt = /^Queue dispatch attempt ([1-3]); base ([0-9a-f]{40})\.$/.exec(previous.description || '');
      const created = Date.parse(previous.created_at);
      if (!receipt || previous.state !== 'pending' || !Number.isFinite(created) || created > now()) {
        throw new Error('Invalid queue dispatch reservation');
      }
      if (now() - created < REGISTRATION_GRACE_MS) continue;
      attempt = Number(receipt[1]) + 1;
      if (attempt > MAX_DISPATCH_ATTEMPTS) {
        failures.push(`${workflow}: dispatch registration retries exhausted`);
        continue;
      }
    }
    if (!await current()) return;
    // Serialize this writer in CI Check and scheduled recovery. Reserve before
    // dispatch so response loss or slow run registration cannot cause a storm.
    await github.rest.repos.createCommitStatus({...context.repo, sha: group.head,
      context: statusContext, state: 'pending',
      description: `Queue dispatch attempt ${attempt}; base ${base}.`});
    if (!await current()) return;
    try {
      await github.rest.actions.createWorkflowDispatch({...context.repo,
        workflow_id: path.basename(workflow), ref: group.branch,
        inputs: {queue_head_sha: group.head, queue_base_ref: group.baseRef,
          queue_base_sha: base}});
    } catch (error) {
      failures.push(`${workflow}: ${error.message}`);
    }
  }
  if (failures.length) throw new Error(failures.join('; '));
  core.info(`Queue candidate ${group.head}: missing CI dispatch reconciled.`);
}

async function dispatchCandidates({github, context, root, core,
  loadInventory = loadBaseInventory, now = () => Date.now()}) {
  const failures = [];
  for (const baseRef of await enabledBranches({github, context, root})) {
    if (!await queueEnabled({github, context, branch: baseRef, root})) continue;
    const entries = await readQueue(github, context, baseRef);
    const heads = new Set(entries.map(entry => entry.headCommit?.oid).filter(Boolean));
    if (!heads.size) continue; // GitHub may not have built an enqueued entry yet.
    const refs = await github.paginate(github.rest.git.listMatchingRefs, {
      ...context.repo, ref: `heads/gh-readonly-queue/${baseRef}/`, per_page: 100,
    });
    if (!Array.isArray(refs) || refs.length >= 1000) throw new Error('Incomplete queue ref discovery');
    for (const head of heads) {
      const candidates = refs.filter(ref => ref.object?.type === 'commit' &&
        ref.object.sha === head && ref.ref?.startsWith('refs/heads/') &&
        QUEUE_REF.exec(ref.ref.slice(11))?.[1] === baseRef);
      if (!candidates.length) continue; // Ref creation/deletion can race discovery.
      if (candidates.length !== 1) throw new Error('Ambiguous queue candidate ref');
      const group = {head, branch: candidates[0].ref.slice(11), baseRef};
      try {
        await dispatchGroup({github, context, group, root, core, loadInventory, now});
        // Recovery does not depend on workflow_run delivery. This trusted
        // scheduled/event-driven controller can independently publish evidence.
        await reconcile({github, context, group, root, core, loadInventory});
      } catch (error) {
        failures.push(`${head}: ${error.message}`);
      }
    }
  }
  if (failures.length) throw new Error(failures.join('; '));
}

async function reconcileQueue(options) {
  const errors = [];
  try { await sweep(options); } catch (error) { errors.push(error.message); }
  // A failed admission API call for one PR must not strand other candidates.
  // Candidate inspection independently fails closed on any unresolved holds.
  try { await dispatchCandidates(options); } catch (error) { errors.push(error.message); }
  if (errors.length) throw new Error(errors.join('; '));
}

module.exports = {dispatchGroup, dispatchCandidates, reconcileQueue};
