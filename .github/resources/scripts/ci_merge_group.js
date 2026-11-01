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
const SHA = /^[0-9a-f]{40}$/;

function candidate(context) {
  const run = context.payload.workflow_run;
  if (context.eventName !== 'workflow_run' || run?.event !== 'merge_group' ||
      run.head_repository?.full_name !== `${context.repo.owner}/${context.repo.repo}` ||
      !/^gh-readonly-queue\/master\/pr-[1-9][0-9]*-[A-Za-z0-9-]+$/.test(run.head_branch || '') ||
      !SHA.test(run.head_sha || '')) return null;
  return {head: run.head_sha, branch: run.head_branch};
}

async function readRef(github, context, branch) {
  const {data} = await github.rest.git.getRef({...context.repo, ref: `heads/${branch}`});
  if (data.ref !== `refs/heads/${branch}` || data.object?.type !== 'commit' ||
      !SHA.test(data.object.sha || '')) throw new Error('Invalid queue or master ref response');
  return data.object.sha;
}

function expectedWorkflows(inventory, workflowFiles) {
  validateInventory(inventory, workflowFiles);
  const expected = [];
  for (const workflow of inventory.workflows) {
    const paused = workflow.path === '.github/workflows/upgrade-test.yml' &&
      workflow.disabled_for_migration === true;
    let masterPR = false;
    if (workflow.pull_request !== null) {
      // Validate the entire PR trigger, but deliberately ignore path filters:
      // the queue tests the combined candidate through every master lane.
      applicable(workflow.pull_request, 'master', []);
      const {paths, 'paths-ignore': ignoredPaths, ...branches} = workflow.pull_request;
      masterPR = applicable(branches, 'master', []);
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
      queueLane = applicable(branches, 'master', []);
    }
    if (paused) continue;
    if (masterPR && !queueLane) throw new Error(`Master CI lane lacks merge_group coverage: ${workflow.path}`);
    if (queueLane) expected.push(workflow.path);
  }
  if (!expected.length) throw new Error('No expected merge_group workflows');
  return expected;
}

async function evidence(github, context, group, expected) {
  const runs = await github.paginate(github.rest.actions.listWorkflowRunsForRepo, {
    ...context.repo, event: 'merge_group', head_sha: group.head, per_page: 100,
  });
  if (!Array.isArray(runs) || runs.length >= 1000) throw new Error('Incomplete queue workflow history');
  const reasons = [];
  let failed = false;
  for (const path of expected) {
    const matching = runs.filter(run => run.path === path && run.event === 'merge_group' &&
      run.head_sha === group.head && run.head_branch === group.branch &&
      run.head_repository?.full_name === `${context.repo.owner}/${context.repo.repo}`);
    for (const run of matching) {
      const invalid = invalidRunMetadata(run);
      if (invalid) throw new Error(`Invalid queue workflow attempt ${invalid}`);
    }
    const attemptTime = run => Math.max(Date.parse(run.created_at), Date.parse(run.run_started_at) || 0);
    matching.sort((a, b) => attemptTime(b) - attemptTime(a) || b.id - a.id);
    const run = matching[0];
    if (!run) reasons.push(`${path}: expected workflow has not registered`);
    else if (run.status !== 'completed' || run.conclusion !== 'success') {
      reasons.push(`${path}: latest run is ${run.status}/${run.conclusion}`);
      if (run.status === 'completed') failed = true;
    }
  }
  return {passed: reasons.length === 0, failed, reasons};
}

async function reconcile({github, context, core, root = process.env.GITHUB_WORKSPACE,
  loadInventory = loadBaseInventory}) {
  const group = candidate(context);
  if (!group) return;
  const publish = async (state, description) => github.rest.repos.createCommitStatus({
    ...context.repo, sha: group.head, context: 'ci-passed', state,
    description: description.slice(0, 140),
    target_url: `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`,
  });
  // The workflow_run publisher runs trusted default-branch code. The candidate
  // ref is only read through APIs; no candidate code or artifact is executed.
  await publish('pending', 'Merge-group CI evidence is being revalidated.');
  try {
    if (await readRef(github, context, group.branch) !== group.head) return;
    const base = await readRef(github, context, 'master');
    const inventory = await loadInventory({github, ...context.repo, root,
      pullRequest: {base: {sha: base, ref: 'master', repo: {full_name:
        `${context.repo.owner}/${context.repo.repo}`}}}});
    const expected = expectedWorkflows(inventory.inventory, inventory.workflowFiles);
    const current = async () => (await readRef(github, context, group.branch)) === group.head &&
      (await readRef(github, context, 'master')) === base;
    let result = await evidence(github, context, group, expected);
    core.info(JSON.stringify({head: group.head, base, expected, ...result}));
    if (!result.passed) {
      if (result.failed) await publish('failure', result.reasons.join('; '));
      return;
    }
    // Reread current attempts and refs immediately before and after publishing.
    // Status writes and queue rebuilds are not atomic; stale evidence must not
    // carry over to another SHA or survive a detected rerun.
    result = await evidence(github, context, group, expected);
    if (!result.passed || !await current()) return;
    await publish('success', 'All expected master merge-group workflows passed.');
    let queueHead;
    try {
      queueHead = await readRef(github, context, group.branch);
    } catch (error) {
      // Success can immediately merge the group and delete its temporary ref.
      // A removed group is inactive; its status cannot approve a rebuilt SHA.
      if (error.status === 404) return;
      throw error;
    }
    if (queueHead !== group.head || await readRef(github, context, 'master') !== base ||
        !(await evidence(github, context, group, expected)).passed) {
      await publish('pending', 'Queue candidate or CI changed; waiting for current evidence.');
    }
  } catch (error) {
    await publish('failure', 'Cannot verify merge-group CI; inspect CI Check and retry.');
    throw error;
  }
}

module.exports = {candidate, expectedWorkflows, evidence, reconcile};
