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

const {verifyExpectedWorkflows, loadLocalInventory} = require('./ci_expected_workflows');

function eligible(pr) {
  const labels = new Set(pr.labels.map(label => label.name));
  return !labels.has('needs-ok-to-test') && (labels.has('ok-to-test') ||
    pr.user.login === 'dependabot[bot]' ||
    ['MEMBER', 'OWNER', 'COLLABORATOR'].includes(pr.author_association));
}

function snapshot(pr) {
  return JSON.stringify([pr.number, pr.state, pr.head.sha, pr.base.ref, pr.base.sha, eligible(pr)]);
}

async function readPR(github, context, number) {
  return (await github.rest.pulls.get({...context.repo, pull_number: number})).data;
}

async function recoveryCandidates({github, context}) {
  const prs = await github.paginate(github.rest.pulls.list, {
    ...context.repo, state: 'open', per_page: 100,
  });
  const candidates = [];
  for (const pr of prs) {
    if (!eligible(pr)) continue;
    // Only recovery needs a timer. Event-driven reconciliation invalidates
    // existing success; avoid allocating a runner for every green PR.
    const {data} = await github.rest.repos.getCombinedStatusForRef({
      ...context.repo, ref: pr.head.sha, per_page: 100,
    });
    if (data.statuses.some(status => status.context === 'ci-passed' && status.state === 'success')) continue;
    candidates.push({number: pr.number, head: pr.head.sha});
  }
  if (candidates.length > 256) throw new Error('Recovery exceeds matrix limit; inspect CI Check.');
  return candidates;
}

async function resolve(github, context, recovery) {
  if (context.eventName === 'schedule') {
    if (!Number.isSafeInteger(recovery?.number) || recovery.number <= 0 || !recovery.head) {
      throw new Error('Scheduled recovery requires a PR number and head SHA');
    }
    const pr = await readPR(github, context, recovery.number);
    return pr.state === 'open' && pr.head.sha === recovery.head ? pr : null;
  }
  if (context.eventName === 'pull_request_target') {
    const eventPR = context.payload.pull_request;
    const pr = await readPR(github, context, eventPR.number);
    // Old events must never publish onto a newer head.
    return pr.head.sha === eventPR.head.sha ? pr : null;
  }
  const run = context.payload.workflow_run;
  if (run?.event !== 'pull_request' || !run.head_repository?.owner?.login || !run.head_branch) {
    return null;
  }
  const pulls = await github.paginate(github.rest.pulls.list, {
    ...context.repo, state: 'open',
    head: `${run.head_repository.owner.login}:${run.head_branch}`, per_page: 100,
  });
  const matches = pulls.filter(pr => pr.head.sha === run.head_sha &&
    pr.head.repo?.full_name === run.head_repository.full_name);
  if (matches.length === 0) return null;
  if (matches.length !== 1) throw new Error('Workflow head must identify exactly one open PR');
  const pr = await readPR(github, context, matches[0].number);
  return pr.head.sha === run.head_sha &&
    pr.head.repo?.full_name === run.head_repository.full_name ? pr : null;
}

async function publish(github, context, pr, state, description) {
  await github.rest.repos.createCommitStatus({
    ...context.repo, sha: pr.head.sha, context: 'ci-passed', state,
    description: description.slice(0, 140),
    target_url: `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`,
  });
  if (state === 'success') {
    await github.rest.issues.addLabels({...context.repo, issue_number: pr.number, labels: ['ci-passed']});
  } else {
    try {
      await github.rest.issues.removeLabel({...context.repo, issue_number: pr.number, name: 'ci-passed'});
    } catch (error) {
      if (error.status !== 404) throw error;
    }
  }
}

async function freshAfter(github, context, pr) {
  // Read durable history on EVERY reconciliation. A queued edited event may
  // be replaced by another event; a label or rerun must not erase retargeting.
  const events = await github.paginate(github.rest.issues.listEventsForTimeline, {
    ...context.repo, issue_number: pr.number, per_page: 100,
  });
  let cutoff = null;
  for (const event of events) {
    if (['base_ref_changed', 'automatic_base_change_succeeded'].includes(event.event)) {
      if (!Number.isFinite(Date.parse(event.created_at))) throw new Error('Base change has no timestamp');
      if (!cutoff || Date.parse(event.created_at) > Date.parse(cutoff)) cutoff = event.created_at;
    }
  }
  return cutoff;
}

async function evidence(github, context, pr, inventory) {
  return verifyExpectedWorkflows({github, ...context.repo, pullRequest: pr,
    ...inventory, freshAfter: await freshAfter(github, context, pr)});
}

async function prepare({github, context, core, recovery, root = process.env.GITHUB_WORKSPACE}) {
  let pr;
  try {
    pr = await resolve(github, context, recovery);
  } catch (error) {
    const sha = context.payload.pull_request?.head.sha || context.payload.workflow_run?.head_sha || recovery?.head;
    if (sha) await github.rest.repos.createCommitStatus({
      ...context.repo, sha, context: 'ci-passed', state: 'failure',
      description: 'Cannot identify the PR for this head; inspect CI Check and retry.',
    });
    throw error;
  }
  if (!pr) return;
  // Set the identity before any further API operation so the always() step
  // can fail closed if inventory loading or evidence retrieval fails.
  core.setOutput('pr_number', String(pr.number));
  core.setOutput('head_sha', pr.head.sha);
  core.setOutput('snapshot', snapshot(pr));
  await publish(github, context, pr, 'pending', 'CI evidence is being revalidated.');
  if (pr.state !== 'open' || !eligible(pr)) return;
  const result = await evidence(github, context, pr, loadLocalInventory(root));
  core.info(JSON.stringify(result));
  core.setOutput('ready', String(result.passed));
}

async function finalize({github, context, core, number, head, before, pollPassed,
  root = process.env.GITHUB_WORKSPACE}) {
  if (!number || !head) return;
  const pr = await readPR(github, context, Number(number));
  const original = {...pr, head: {...pr.head, sha: head}};
  let passed = false;
  let reason = 'CI did not pass; complete current-head CI and retry.';
  try {
    if (pr.head.sha === head && pr.state === 'open' && snapshot(pr) === before && eligible(pr) && pollPassed) {
      const result = await evidence(github, context, pr, loadLocalInventory(root));
      core.info(JSON.stringify(result));
      passed = result.passed;
      if (!passed) reason = result.reasons.join('; ');
    }
    await publish(github, context, original, passed ? 'success' : 'failure',
      passed ? 'Expected CI and all checks passed for this head.' : reason);
    // Status/label writes are not atomic with PR updates. Re-read the full
    // state and durable base history, and undo success when either drifted.
    if (passed) {
      const after = await readPR(github, context, Number(number));
      const current = snapshot(after) === before &&
        (await evidence(github, context, after, loadLocalInventory(root))).passed;
      if (!current) await publish(github, context, original, 'failure',
        'PR or CI changed during publication; rerun CI on the current head.');
    }
  } catch (error) {
    await publish(github, context, original, 'failure', 'Cannot verify CI evidence; inspect CI Check and retry.');
    throw error;
  }
}

module.exports = {recoveryCandidates, eligible, snapshot, resolve, freshAfter, prepare, finalize};
