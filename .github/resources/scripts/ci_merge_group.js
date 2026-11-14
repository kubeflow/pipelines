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

const {applicable, validateInventory, invalidRunMetadata, loadBaseInventory} = require('./ci_expected_workflows');
const {inspectQueue} = require('./merge_queue_admission');
const {queueEnabled} = require('./merge_queue_policy');
const SHA = /^[0-9a-f]{40}$/;

function candidate(context) {
  const run = context.payload.workflow_run;
  const queue = /^gh-readonly-queue\/(master|main)\/pr-[1-9][0-9]*-[A-Za-z0-9-]+$/.exec(run?.head_branch || '');
  if (context.eventName !== 'workflow_run' || !['merge_group', 'workflow_dispatch'].includes(run?.event) ||
      run.head_repository?.full_name !== `${context.repo.owner}/${context.repo.repo}` ||
      !queue ||
      !SHA.test(run.head_sha || '')) return null;
  return {head: run.head_sha, branch: run.head_branch, baseRef: queue[1]};
}

async function readRef(github, context, branch) {
  const {data} = await github.rest.git.getRef({...context.repo, ref: `heads/${branch}`});
  if (data.ref !== `refs/heads/${branch}` || data.object?.type !== 'commit' ||
      !SHA.test(data.object.sha || '')) throw new Error('Invalid queue or base ref response');
  return data.object.sha;
}

function expectedWorkflows(inventory, workflowFiles, baseRef) {
  validateInventory(inventory, workflowFiles);
  const expected = [];
  for (const workflow of inventory.workflows) {
    const paused = workflow.path === '.github/workflows/upgrade-test.yml' &&
      workflow.disabled_for_migration === true;
    let basePR = false;
    if (workflow.pull_request !== null) {
      // Validate the entire PR trigger, but deliberately ignore path filters:
      // the queue tests the combined candidate through every lane for its target branch.
      applicable(workflow.pull_request, baseRef, []);
      const {paths, 'paths-ignore': ignoredPaths, ...branches} = workflow.pull_request;
      basePR = applicable(branches, baseRef, []);
    }
    const trigger = workflow.merge_group;
    let queueLane = false;
    if (trigger !== null && trigger !== undefined) {
      if (typeof trigger !== 'object' || Array.isArray(trigger) ||
          Object.keys(trigger).some(key => !['types', 'branches', 'branches-ignore'].includes(key))) {
        throw new Error(`Unsupported merge_group trigger: ${workflow.path}`);
      }
      const {types, ...branches} = trigger;
      if (types !== undefined && (!Array.isArray(types) || types.length !== 1 ||
          types[0] !== 'checks_requested')) throw new Error('Unsupported merge_group activity types');
      queueLane = applicable(branches, baseRef, []);
    }
    if (paused) continue;
    if (basePR && !queueLane) throw new Error(`${baseRef} CI lane lacks merge_group coverage: ${workflow.path}`);
    if (queueLane) expected.push(workflow.path);
  }
  if (!expected.length) throw new Error('No expected merge_group workflows');
  return expected;
}

async function queueRuns(github, context, group) {
  const runs = [];
  for (const event of ['merge_group', 'workflow_dispatch']) {
    const page = await github.paginate(github.rest.actions.listWorkflowRunsForRepo, {
      ...context.repo, event, head_sha: group.head, per_page: 100,
    });
    if (!Array.isArray(page) || page.length >= 1000) throw new Error('Incomplete queue workflow history');
    runs.push(...page);
  }
  return runs;
}

function matchingRuns(runs, context, group, path) {
  const matching = runs.filter(run => run.path === path &&
    ['merge_group', 'workflow_dispatch'].includes(run.event) &&
    run.head_sha === group.head && run.head_branch === group.branch &&
    run.head_repository?.full_name === `${context.repo.owner}/${context.repo.repo}`);
  for (const run of matching) {
    const invalid = invalidRunMetadata(run);
    if (invalid) throw new Error(`Invalid queue workflow attempt ${invalid}`);
  }
  const attemptTime = run => Math.max(Date.parse(run.created_at), Date.parse(run.run_started_at) || 0);
  return matching.sort((a, b) => attemptTime(b) - attemptTime(a) || b.id - a.id);
}

async function triggeringRun(github, context, group) {
  const eventGroup = candidate(context);
  if (!eventGroup || eventGroup.head !== group.head || eventGroup.branch !== group.branch) return null;
  const event = context.payload.workflow_run;
  if (!Number.isSafeInteger(event?.id) || event.id <= 0 ||
      !Number.isSafeInteger(event.run_attempt) || event.run_attempt <= 0) {
    throw new Error('Invalid triggering queue workflow identity');
  }
  const {data: run} = await github.rest.actions.getWorkflowRun({...context.repo, run_id: event.id});
  if (invalidRunMetadata(run) || run.id !== event.id || run.run_attempt < event.run_attempt ||
      run.event !== event.event || run.path !== event.path ||
      run.repository?.full_name !== `${context.repo.owner}/${context.repo.repo}` ||
      run.head_repository?.full_name !== `${context.repo.owner}/${context.repo.repo}` ||
      run.head_sha !== group.head || run.head_branch !== group.branch) {
    throw new Error('Triggering queue workflow identity changed or is incomplete');
  }
  return run;
}

async function evidence(github, context, group, expected, allowDispatch = false) {
  const runs = (await queueRuns(github, context, group))
    .filter(run => allowDispatch || run.event === 'merge_group');
  const trigger = await triggeringRun(github, context, group);
  const reasons = [];
  let failed = false;
  for (const path of expected) {
    const run = matchingRuns(runs, context, group, path)[0];
    // A workflow_run event can arrive before its newer attempt appears in the
    // list endpoint. The direct lookup establishes a floor for that execution;
    // repeated stale list reads must never restore its earlier green result.
    if (trigger?.path === path && run &&
        (run.id === trigger.id || matchingRuns([run, trigger], context, group, path)[0] === trigger)) {
      if (run.id !== trigger.id || run.run_attempt < trigger.run_attempt ||
          run.status !== trigger.status || run.conclusion !== trigger.conclusion) {
        reasons.push(`${path}: awaiting current triggering workflow attempt`);
        if (trigger.status === 'completed' && trigger.conclusion !== 'success') failed = true;
        continue;
      }
    }
    if (!run) reasons.push(`${path}: expected workflow has not registered`);
    else if (run.status !== 'completed' || run.conclusion !== 'success') {
      reasons.push(`${path}: latest run is ${run.status}/${run.conclusion}`);
      if (run.status === 'completed') failed = true;
    }
  }
  return {passed: reasons.length === 0, failed, reasons};
}

async function reconcile({github, context, core, root = process.env.GITHUB_WORKSPACE,
  loadInventory = loadBaseInventory, group: requestedGroup}) {
  const group = requestedGroup || candidate(context);
  if (!group) return;
  const publish = async (state, description) => github.rest.repos.createCommitStatus({
    ...context.repo, sha: group.head, context: 'ci-passed', state,
    description: description.slice(0, 140),
    target_url: `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`,
  });
  // The workflow_run publisher runs trusted default-branch code. The candidate
  // ref is only read through APIs; no candidate code or artifact is executed.
  try {
    const allowDispatch = await queueEnabled({github, context, branch: group.baseRef, root});
    if ((requestedGroup || context.payload.workflow_run?.event === 'workflow_dispatch') && !allowDispatch) return;
    await publish('pending', 'Merge-group CI evidence is being revalidated.');
    if (await readRef(github, context, group.branch) !== group.head) return;
    const base = await readRef(github, context, group.baseRef);
    const inventory = await loadInventory({github, ...context.repo, root,
      pullRequest: {base: {sha: base, ref: group.baseRef, repo: {full_name:
        `${context.repo.owner}/${context.repo.repo}`}}}});
    const expected = expectedWorkflows(inventory.inventory, inventory.workflowFiles, group.baseRef);
    const policy = await inspectQueue({github, context, group, root});
    if (allowDispatch && policy === null) {
      throw new Error('Queue policy disabled before candidate validation.');
    }
    const current = async () => (await readRef(github, context, group.branch)) === group.head &&
      (await readRef(github, context, group.baseRef)) === base &&
      (await inspectQueue({github, context, group, root})) === policy;
    let result = await evidence(github, context, group, expected, allowDispatch);
    core.info(JSON.stringify({head: group.head, base, expected, ...result}));
    if (!result.passed) {
      if (result.failed) await publish('failure', result.reasons.join('; '));
      return;
    }
    // Reread current attempts and refs immediately before and after publishing.
    // Status writes and queue rebuilds are not atomic; stale evidence must not
    // carry over to another SHA or survive a detected rerun.
    result = await evidence(github, context, group, expected, allowDispatch);
    if (!result.passed || !await current()) return;
    await publish('success', `All expected ${group.baseRef} merge-group workflows passed.`);
    let queueHead;
    try {
      queueHead = await readRef(github, context, group.branch);
    } catch (error) {
      // Success can immediately merge the group and delete its temporary ref.
      // A removed group is inactive; its status cannot approve a rebuilt SHA.
      if (error.status === 404) return;
      throw error;
    }
    if (queueHead !== group.head || await readRef(github, context, group.baseRef) !== base ||
        !(await evidence(github, context, group, expected, allowDispatch)).passed ||
        (await inspectQueue({github, context, group, root})) !== policy) {
      await publish('pending', 'Queue candidate or CI changed; waiting for current evidence.');
    }
  } catch (error) {
    await publish('failure', 'Cannot verify merge-group CI; inspect CI Check and retry.');
    throw error;
  }
}

module.exports = {candidate, readRef, expectedWorkflows, queueRuns, matchingRuns, evidence, reconcile};
