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

const {verifyExpectedWorkflows, loadBaseInventory} = require('./ci_expected_workflows');
const {verifyCheckRuns} = require('./ci_check_runs');

function blocked(pr) {
  // Tide's human-PR queries still rely on this explicit merge hold.
  return pr.labels.some(label => label.name === 'needs-ok-to-test');
}

function snapshot(pr) {
  return JSON.stringify([pr.number, pr.state, pr.head.sha, pr.base.ref, pr.base.sha, blocked(pr)]);
}

async function readPR(github, context, number) {
  return (await github.rest.pulls.get({...context.repo, pull_number: number})).data;
}

async function currentStatus(github, context, head) {
  // Combined-status responses are paginated independently of commit history.
  for await (const response of github.paginate.iterator(github.rest.repos.getCombinedStatusForRef, {
    ...context.repo, ref: head, per_page: 100,
  })) {
    const statuses = response.data.statuses || response.data;
    const status = statuses.find(item => item.context === 'ci-passed');
    if (status) return status;
  }
  return null;
}

async function baseRevision(github, context, pr) {
  // ONE validated base revision for policy loading, execution freshness,
  // publication, and the post-publication check. GitHub freezes pr.base.sha at
  // the PR's last sync, so any use of the frozen sha leaves a stale green
  // untouched as the base advances. Key every use off the LIVE base branch tip
  // instead, so an advance revokes the green and forces fresh CI against the
  // new base. Release branches keep their frozen base.sha behavior.
  let sha = pr.base.sha;
  if (pr.base.ref === 'master') {
    const {data} = await github.rest.git.getRef({
      ...context.repo, ref: `heads/${pr.base.ref}`,
    });
    sha = data.object.sha;
  }
  return sha;
}

async function basePolicyStamp(github, context, pr) {
  const sha = await baseRevision(github, context, pr);
  return require('node:crypto').createHash('sha256')
    .update(JSON.stringify([pr.base.ref, sha])).digest('hex');
}

async function successDescription(github, context, pr) {
  // Bind green evidence to the exact checked-in workflow policy. Legacy
  // statuses and statuses from another base must be reconsidered by recovery.
  const stamp = await basePolicyStamp(github, context, pr);
  return `Expected CI and all checks passed; base policy ${stamp}.`;
}

async function recoveryCandidates({github, context}) {
  const prs = await github.paginate(github.rest.pulls.list, {
    ...context.repo, state: 'open', per_page: 100,
  });
  const candidates = [];
  for (const pr of prs) {
    if (blocked(pr)) continue;
    // Revisit green heads when their trusted base policy changes, including
    // statuses published before base-policy stamps were introduced.
    const status = await currentStatus(github, context, pr.head.sha);
    if (status?.state === 'success' &&
        status.description === await successDescription(github, context, pr)) continue;
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

async function publish(github, context, pr, state, description, provisional = false) {
  let current;
  try {
    current = await currentStatus(github, context, pr.head.sha);
  } catch (error) {
    // A failed read must not prevent invalidating a previously green head.
    if (state === 'success') throw error;
  }
  // Revoke green immediately without creating two statuses for every event
  // on an unchanged pending/failing head. Authoritative evidence may still
  // move an earlier failure to pending when its rerun starts.
  const preserve = provisional && current && current.state !== 'success';
  const boundedDescription = description.slice(0, 140);
  if (!preserve && (current?.state !== state || current.description !== boundedDescription)) {
    await github.rest.repos.createCommitStatus({
      ...context.repo, sha: pr.head.sha, context: 'ci-passed', state,
      description: boundedDescription,
      target_url: `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`,
    });
  }
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

async function evidence(github, context, pr, root) {
  const baseSha = await baseRevision(github, context, pr);
  // Only the live master tip advances behind the publisher's back; release
  // branches keep their frozen base.sha behavior (see baseRevision). The
  // freshness check is scoped to the live base so release branches are
  // unchanged.
  const liveBase = pr.base.ref === 'master';
  const [inventory, cutoff] = await Promise.all([
    loadBaseInventory({github, ...context.repo, pullRequest: pr, baseSha, root}),
    freshAfter(github, context, pr),
  ]);
  const args = {github, ...context.repo, pullRequest: pr, baseSha, liveBase, ...inventory, freshAfter: cutoff};
  let result = await verifyExpectedWorkflows(args);
  if (result.missing.length) {
    // The earliest publication on this SHA starts the registration grace.
    // Labels, retries, and changing explanations cannot restart that clock.
    const statuses = await github.paginate(github.rest.repos.listCommitStatusesForRef, {
      ...context.repo, ref: pr.head.sha, per_page: 100,
    });
    const times = statuses.filter(status => status.context === 'ci-passed').map(status => {
      const time = Date.parse(status.created_at);
      if (!Number.isFinite(time)) throw new Error('CI status has no registration timestamp');
      return time;
    });
    if (times.length) {
      const start = Math.max(Math.min(...times), cutoff ? Date.parse(cutoff) : 0);
      result = await verifyExpectedWorkflows({...args, registrationStartedAt: new Date(start).toISOString()});
    }
  }
  return result;
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
  await publish(github, context, pr, 'pending', 'CI evidence is being revalidated.', true);
  if (pr.state !== 'open' || blocked(pr)) return;
  const result = await evidence(github, context, pr, root);
  core.info(JSON.stringify(result));
  core.setOutput('ready', String(result.passed));
}

async function finalize({github, context, core, number, head, before, pollPassed, pollSkipped = false,
  root = process.env.GITHUB_WORKSPACE}) {
  if (!number || !head) return;
  let original = {number: Number(number), head: {sha: head}};
  let errorReason = 'Cannot verify CI evidence; inspect CI Check and retry.';
  try {
    const pr = await readPR(github, context, Number(number));
    original = {...pr, head: {...pr.head, sha: head}};
    let state = 'failure';
    let reason = blocked(pr) ? 'PR is held by needs-ok-to-test; obtain maintainer approval.' :
      'PR changed or is closed; complete current-head CI and retry.';
    if (pr.head.sha === head && pr.state === 'open' && snapshot(pr) === before && !blocked(pr)) {
      const [workflows, checks] = await Promise.all([
        evidence(github, context, pr, root),
        verifyCheckRuns({github, ...context.repo, sha: head}),
      ]);
      core.info(JSON.stringify({workflows, checks}));
      const results = [workflows, checks];
      state = results.some(result => result.state === 'failure') ? 'failure' :
        results.some(result => result.state === 'pending') ? 'pending' : 'success';
      reason = results.filter(result => result.state === state)
        .flatMap(result => result.reasons).join('; ');
      // Retain the pinned checker's independent success requirement. If it
      // was skipped before workflow evidence recovered, wait for the next
      // reconciliation instead of authorizing an unchecked success.
      if (state === 'success' && !pollPassed) {
        state = pollSkipped ? 'pending' : 'failure';
        reason = pollSkipped ? 'CI changed since initial assessment; awaiting check validation.' :
          'Cannot verify all checks passed; inspect CI Check and retry.';
      }
    }
    if (state === 'failure') errorReason = reason;
    await publish(github, context, original, state,
      state === 'success' ? await successDescription(github, context, pr) : reason);
    // Status/label writes are not atomic with PR or CI changes. Revalidate
    // external checks as well as workflow evidence after publishing green.
    if (state === 'success') {
      const after = await readPR(github, context, Number(number));
      if (snapshot(after) !== before) {
        errorReason = 'PR changed during publication; rerun CI on the current head.';
        await publish(github, context, original, 'failure', errorReason);
      } else {
        const results = await Promise.all([
          evidence(github, context, after, root),
          verifyCheckRuns({github, ...context.repo, sha: head}),
        ]);
        const changed = results.find(result => result.state === 'failure') ||
          results.find(result => result.state === 'pending');
        if (changed) {
          if (changed.state === 'failure') errorReason = changed.reasons.join('; ');
          await publish(github, context, original, changed.state, changed.reasons.join('; '));
        }
      }
    }
  } catch (error) {
    await publish(github, context, original, 'failure', errorReason);
    throw error;
  }
}

module.exports = {recoveryCandidates, snapshot, resolve, freshAfter, prepare, finalize};
