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
const {readQueue, inspectDispatchSnapshot, sweep} = require('./merge_queue_admission');
const {expectedWorkflows, queueRuns, matchingRuns, reconcile, candidate} = require('./ci_merge_group');
const {loadBaseInventory, loadLocalInventory, invalidRunMetadata} = require('./ci_expected_workflows');
const QUEUE_REF = /^gh-readonly-queue\/(master|main|release-2\.18)\/pr-[1-9][0-9]*-[A-Za-z0-9-]+$/;
const SHA = /^[0-9a-f]{40}$/;
const REGISTRATION_GRACE_MS = 120000;
const MAX_DISPATCH_ATTEMPTS = 3;
const DISPATCH_CONCURRENCY = 4;

// GitHub normally starts queue CI through merge_group. In our integration probes,
// admission using GITHUB_TOKEN created the candidate but did not start that CI.
// workflow_dispatch is an explicit exception to GITHUB_TOKEN event suppression;
// use it for missing candidate runs while retaining native-run deduplication.
// This workaround is specific to builtin-token admission, not a queue requirement.
// Evidence and removal considerations: docs/agents/merge-queue-activation.md.
async function dispatchGroup({github, context, group, core, root = process.env.GITHUB_WORKSPACE,
  loadInventory = loadBaseInventory, now = () => Date.now()}) {
  const ref = QUEUE_REF.exec(group.branch || '');
  if (!ref || ref[1] !== group.baseRef || !SHA.test(group.head || '')) {
    throw new Error('Invalid queue dispatch candidate');
  }
  const initial = await inspectDispatchSnapshot({github, context, group, root});
  if (initial === null || initial.head !== group.head) return;
  const {base, policy} = initial;
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
  const current = async () => {
    const snapshot = await inspectDispatchSnapshot({github, context, group, root});
    return snapshot !== null && snapshot.head === group.head && snapshot.base === base &&
      snapshot.policy === policy;
  };
  const failures = [];
  const dispatchWorkflow = async workflow => {
    // Registered attempts, including failures/cancellations, are never silently
    // rerun. Latest evidence is handled by the publisher; deliberate retries use
    // Actions' rerun UI. The reservation handles only missing dispatch responses.
    if (matchingRuns(runs, context, group, workflow).length) return true;
    const statusContext = `ci-queue-dispatch/${path.basename(workflow)}`;
    const previous = statuses.find(status => status.context === statusContext);
    let attempt = 1;
    if (previous) {
      const receipt = /^Queue dispatch attempt ([1-3]); base ([0-9a-f]{40})\.$/.exec(previous.description || '');
      const created = Date.parse(previous.created_at);
      if (!receipt || previous.state !== 'pending' || !Number.isFinite(created) || created > now()) {
        throw new Error('Invalid queue dispatch reservation');
      }
      if (now() - created < REGISTRATION_GRACE_MS) return true;
      attempt = Number(receipt[1]) + 1;
      if (attempt > MAX_DISPATCH_ATTEMPTS) {
        failures.push(`${workflow}: dispatch registration retries exhausted`);
        return true;
      }
    }
    if (!await current()) return false;
    // CI Check and recovery serialize controller runs. Reserve this lane before
    // dispatch so response loss or slow run registration cannot cause a storm.
    await github.rest.repos.createCommitStatus({...context.repo, sha: group.head,
      context: statusContext, state: 'pending',
      description: `Queue dispatch attempt ${attempt}; base ${base}.`});
    if (!await current()) return false;
    try {
      await github.rest.actions.createWorkflowDispatch({...context.repo,
        workflow_id: path.basename(workflow), ref: group.branch,
        inputs: {queue_head_sha: group.head, queue_base_ref: group.baseRef,
          queue_base_sha: base}});
    } catch (error) {
      failures.push(`${workflow}: ${error.message}`);
    }
    return true;
  };
  for (let offset = 0; offset < expected.length; offset += DISPATCH_CONCURRENCY) {
    // Each lane validates fresh policy around its own reservation. Bound the
    // fanout and drain every active request before returning or reporting errors.
    const results = await Promise.allSettled(
      expected.slice(offset, offset + DISPATCH_CONCURRENCY).map(dispatchWorkflow));
    const rejected = results.find(result => result.status === 'rejected');
    if (rejected) throw rejected.reason;
    if (results.some(result => result.value === false)) return;
  }
  if (failures.length) throw new Error(failures.join('; '));
  core.info(`Queue candidate ${group.head}: missing CI dispatch reconciled.`);
}

async function dispatchCandidates({github, context, root = process.env.GITHUB_WORKSPACE, core,
  loadInventory = loadBaseInventory, now = () => Date.now()}) {
  const failures = [];
  for (const baseRef of await enabledBranches({github, context, root})) {
    // A broken queue must not prevent other enabled branches from progressing.
    try {
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
    } catch (error) {
      failures.push(`${baseRef}: ${error.message}`);
    }
  }
  if (failures.length) throw new Error(failures.join('; '));
}

// Only a missing/replaced immutable queue ref proves that its CI is obsolete.
// Queue membership and API failures alone are insufficient cancellation evidence.
async function cancelObsoleteCandidates({github, context, core,
  root = process.env.GITHUB_WORKSPACE, branches, workflowPaths}) {
  const trigger = candidate(context);
  if (context.eventName !== 'schedule' && !trigger) return;
  branches ??= await enabledBranches({github, context, root});
  if (!branches.length || (trigger && !branches.includes(trigger.baseRef))) return;
  if (!workflowPaths) {
    const local = loadLocalInventory(root);
    // The validated default-branch inventory includes registration-only release
    // companions. Their real queue triggers live on the release branch.
    workflowPaths = new Set(local.inventory.workflows.map(workflow => workflow.path));
  }
  const active = new Set(['queued', 'in_progress', 'waiting', 'pending', 'requested']);
  const runs = [];
  if (trigger) {
    runs.push(...await queueRuns(github, context, trigger));
  } else {
    // Discover only active work, rather than paging through historical builds.
    // A saturated API result cannot establish complete discovery.
    for (const status of active) {
      const page = await github.paginate(github.rest.actions.listWorkflowRunsForRepo,
        {...context.repo, status, per_page: 100});
      if (!Array.isArray(page) || page.length >= 1000) {
        throw new Error('Incomplete active queue workflow discovery');
      }
      runs.push(...page);
    }
  }
  const eligible = run => {
    const match = QUEUE_REF.exec(run?.head_branch || '');
    return match && branches.includes(match[1]) && workflowPaths.has(run.path) &&
      ['workflow_dispatch', 'merge_group'].includes(run.event) && active.has(run.status) &&
      SHA.test(run.head_sha || '') &&
      run.head_repository?.full_name === `${context.repo.owner}/${context.repo.repo}`;
  };
  const refHead = async branch => {
    let data;
    try {
      ({data} = await github.rest.git.getRef({...context.repo, ref: `heads/${branch}`}));
    } catch (error) {
      if (error.status === 404) return null;
      throw error;
    }
    if (data.ref !== `refs/heads/${branch}` || data.object?.type !== 'commit' ||
        !SHA.test(data.object.sha || '')) throw new Error('Invalid candidate cleanup ref');
    return data.object.sha;
  };
  const heads = new Map();
  const reservations = new Map();
  const seen = new Set();
  const failures = [];
  for (const run of runs) {
    if (!eligible(run) || seen.has(run.id)) continue;
    seen.add(run.id);
    try {
      const invalid = invalidRunMetadata(run);
      if (invalid) throw new Error(`Invalid cleanup workflow attempt ${invalid}`);
      if (!heads.has(run.head_branch)) heads.set(run.head_branch, await refHead(run.head_branch));
      if (heads.get(run.head_branch) === run.head_sha) continue;
      if (run.event === 'workflow_dispatch') {
        // A manual workflow may intentionally inspect a queue ref. Only cancel
        // lanes reserved by our controller before this run was created.
        if (!reservations.has(run.head_sha)) {
          const statuses = await github.paginate(github.rest.repos.listCommitStatusesForRef,
            {...context.repo, ref: run.head_sha, per_page: 100});
          if (!Array.isArray(statuses) || statuses.length >= 1000) {
            throw new Error('Incomplete obsolete queue dispatch reservations');
          }
          reservations.set(run.head_sha, statuses);
        }
        const reserved = reservations.get(run.head_sha).some(status =>
          status.context === `ci-queue-dispatch/${path.basename(run.path)}` &&
          status.state === 'pending' &&
          /^Queue dispatch attempt [1-3]; base [0-9a-f]{40}\.$/.test(status.description || '') &&
          Number.isFinite(Date.parse(status.created_at)) &&
          Date.parse(status.created_at) <= Date.parse(run.created_at));
        if (!reserved) continue;
      }
      const {data: current} = await github.rest.actions.getWorkflowRun({...context.repo, run_id: run.id});
      if (!eligible(current) || current.id !== run.id || current.run_attempt !== run.run_attempt ||
          current.head_branch !== run.head_branch || current.head_sha !== run.head_sha ||
          current.path !== run.path || current.event !== run.event) continue;
      const currentInvalid = invalidRunMetadata(current);
      if (currentInvalid) throw new Error(`Invalid current cleanup workflow attempt ${currentInvalid}`);
      if (await refHead(run.head_branch) === run.head_sha) continue;
      await github.rest.actions.cancelWorkflowRun({...context.repo, run_id: run.id});
      core.info(`Cancelled obsolete queue workflow run ${run.id}.`);
    } catch (error) {
      failures.push(`${run.id}: ${error.message}`);
    }
  }
  if (failures.length) throw new Error(failures.join('; '));
}

async function reconcileQueue(options) {
  const errors = [];
  try { await cancelObsoleteCandidates(options); } catch (error) { errors.push(error.message); }
  try { await sweep(options); } catch (error) { errors.push(error.message); }
  // A failed admission API call for one PR must not strand other candidates.
  // Candidate inspection independently fails closed on any unresolved holds.
  try { await dispatchCandidates(options); } catch (error) { errors.push(error.message); }
  if (errors.length) throw new Error(errors.join('; '));
}

module.exports = {dispatchGroup, dispatchCandidates, cancelObsoleteCandidates, reconcileQueue};
