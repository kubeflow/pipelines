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

function basePolicyStamp(baseRef, baseSha) {
  // Bind green evidence to the exact checked-in workflow policy. The base
  // revision is resolved ONCE per cycle and threaded through every site, so
  // this helper must never re-resolve the live tip (re-resolving can stamp a
  // green with a base whose evidence was validated against another revision).
  return require('node:crypto').createHash('sha256')
    .update(JSON.stringify([baseRef, baseSha])).digest('hex');
}

function successDescription(pr, baseSha) {
  // Legacy statuses and statuses from another base must be reconsidered by
  // recovery. baseSha is the single validated revision resolved by the caller.
  const stamp = basePolicyStamp(pr.base.ref, baseSha);
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
        status.description === successDescription(pr, await baseRevision(github, context, pr))) continue;
    candidates.push({number: pr.number, head: pr.head.sha});
  }
  if (candidates.length > 256) throw new Error('Recovery exceeds matrix limit; inspect CI Check.');
  return candidates;
}

async function revokeStaleGreens({github, context, tip}) {
  // Push-triggered revocation narrows the stale-green merge window (#14762): the
  // moment master advances, a ci-passed success stamped against any prior base
  // revision is revoked instead of waiting for the next scheduled sweep.
  // Freshness reuses the same determination as recovery: a green is fresh only
  // when its description still carries the stamp for the live base tip. This
  // writes only the commit status; the sweep reconciles the informational label
  // with its broader permissions.
  if (!/^[0-9a-f]{40}$/.test(tip || '')) {
    throw new Error('Stale revocation requires a full base commit SHA');
  }
  // Only the most recent push may revoke. Push jobs can execute out of order:
  // master may advance B then C, CI Check publish a success stamped for C, and
  // a delayed B job would otherwise treat that valid C green as stale and
  // overwrite it with failure. Resolve the LIVE master tip and bail out when it
  // has already moved past this push's tip; the newer job (or the sweep) owns
  // revocation for the advanced base.
  const {data: ref} = await github.rest.git.getRef({
    ...context.repo, ref: 'heads/master',
  });
  if (ref.object.sha !== tip) return 0;
  const prs = await github.paginate(github.rest.pulls.list, {
    ...context.repo, state: 'open', base: 'master', per_page: 100,
  });
  let revoked = 0;
  for (const pr of prs) {
    if (blocked(pr)) continue;
    const status = await currentStatus(github, context, pr.head.sha);
    // Revalidate AFTER reading the status: the combined-status lookup can
    // paginate, and master may advance mid-read publishing a fresher success.
    const {data: liveRef} = await github.rest.git.getRef({...context.repo, ref: 'heads/master'});
    if (liveRef.object.sha !== tip) return revoked;
    // Never revoke a head without a ci-passed success.
    if (status?.state !== 'success') continue;
    if (status.description === successDescription(pr, tip)) continue;
    await github.rest.repos.createCommitStatus({
      ...context.repo, sha: pr.head.sha, context: 'ci-passed', state: 'failure',
      description: 'Base advanced; re-test against new master.',
      target_url: `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`,
    });
    revoked++;
  }
  return revoked;
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

async function publish(github, context, pr, state, description, provisional = false, expectedState = null, expectedId = null, updateLabels = true) {
  let current;
  try {
    current = await currentStatus(github, context, pr.head.sha);
  } catch (error) {
    // A failed read must not prevent invalidating a previously green head.
    if (state === 'success') throw error;
  }
  // Explicit merged recovery must not overwrite a status changed by another
  // writer after verification. Ordinary reconciliation keeps its existing rules.
  if ((expectedState !== null && current?.state !== expectedState) ||
      (expectedId !== null && current?.id !== expectedId)) return;
  let written;
  // Revoke green immediately without creating two statuses for every event
  // on an unchanged pending/failing head. Authoritative evidence may still
  // move an earlier failure to pending when its rerun starts.
  const preserve = provisional && current && current.state !== 'success';
  const boundedDescription = description.slice(0, 140);
  if (!preserve && (current?.state !== state || current.description !== boundedDescription)) {
    written = (await github.rest.repos.createCommitStatus({
      ...context.repo, sha: pr.head.sha, context: 'ci-passed', state,
      description: boundedDescription,
      target_url: `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`,
    })).data;
  }
  if (!updateLabels) return written;
  if (state === 'success') {
    await github.rest.issues.addLabels({...context.repo, issue_number: pr.number, labels: ['ci-passed']});
  } else {
    try {
      await github.rest.issues.removeLabel({...context.repo, issue_number: pr.number, name: 'ci-passed'});
    } catch (error) {
      if (error.status !== 404) throw error;
    }
  }
  return written;
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
  return {...result, baseSha};
}

async function prepare({github, context, core, recovery, root = process.env.GITHUB_WORKSPACE}) {
  let pr;
  try {
    pr = await resolve(github, context, recovery);
  } catch (error) {
    const sha = context.payload.pull_request?.head.sha || context.payload.workflow_run?.head_sha || recovery?.head;
    if (sha && context.payload.pull_request?.merged !== true) await github.rest.repos.createCommitStatus({
      ...context.repo, sha, context: 'ci-passed', state: 'failure',
      description: 'Cannot identify the PR for this head; inspect CI Check and retry.',
    });
    throw error;
  }
  if (!pr) return;
  // A merged PR must keep its published status; never publish pending over it.
  // Return before emitting the identity outputs so the always() finalize step
  // is skipped (it bails when number/head are unset) and cannot republish a
  // failure over the merged head.
  if (pr.merged) return;
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
    // Preserve the status on merge; closure without merge still invalidates below.
    if (pr.merged) return;
    let state = 'failure';
    let validatedBaseSha;
    let reason = blocked(pr) ? 'PR is held by needs-ok-to-test; obtain maintainer approval.' :
      'PR changed or is closed; complete current-head CI and retry.';
    if (pr.head.sha === head && pr.state === 'open' && snapshot(pr) === before && !blocked(pr)) {
      const [workflows, checks] = await Promise.all([
        evidence(github, context, pr, root),
        verifyCheckRuns({github, ...context.repo, sha: head}),
      ]);
      validatedBaseSha = workflows.baseSha;
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
      state === 'success' ? successDescription(pr, validatedBaseSha) : reason);
    // Status/label writes are not atomic with PR or CI changes. Revalidate
    // external checks as well as workflow evidence after publishing green.
    if (state === 'success') {
      const after = await readPR(github, context, Number(number));
      // A merge during publication preserves the success that authorized it.
      if (after.merged) return;
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
    // A transient read failure must not flip a merged PR to failure. Re-read
    // once: if the PR merged in the meantime, preserve its status and stop.
    try {
      const merged = await readPR(github, context, Number(number));
      if (merged.merged) return;
    } catch (retryError) {
      // Still unreadable; fall through and fail closed.
    }
    await publish(github, context, original, 'failure', errorReason);
    throw error;
  }
}

// Explicit recovery never changes ordinary PR reconciliation. Only trusted
// default-branch dispatches may revisit a pending aggregate after a release merge.
async function mergedRecoveryBinding(github, context, number, head) {
  const repository = `${context.repo.owner}/${context.repo.repo}`;
  if (context.eventName !== 'workflow_dispatch' ||
      context.payload.repository?.full_name !== repository ||
      context.ref !== `refs/heads/${context.payload.repository.default_branch}` ||
      !Number.isSafeInteger(Number(number)) || Number(number) <= 0 ||
      !/^[0-9a-f]{40}$/.test(head || '')) {
    throw new Error('Merged release recovery requires trusted default-branch dispatch, PR number and exact head.');
  }
  const pr = await readPR(github, context, Number(number));
  if (pr.number !== Number(number) || pr.merged !== true || pr.state !== 'closed' ||
      pr.head.sha !== head || pr.commits !== 1 || !pr.head.repo?.full_name ||
      pr.base.repo?.full_name !== repository || !/^release-\d+\.\d+$/.test(pr.base.ref) ||
      !/^[0-9a-f]{40}$/.test(pr.merge_commit_sha || '') || blocked(pr)) {
    throw new Error('Recovery identity is not an eligible merged release PR.');
  }
  const {data: commit} = await github.rest.repos.getCommit({
    ...context.repo, ref: pr.merge_commit_sha,
  });
  const {data: tested} = await github.rest.repos.getCommit({...context.repo, ref: head});
  if (commit.sha !== pr.merge_commit_sha || tested.sha !== head ||
      !Array.isArray(commit.parents) || commit.parents.length !== 1 ||
      !Array.isArray(tested.parents) || tested.parents.length !== 1 ||
      !/^[0-9a-f]{40}$/.test(commit.parents[0].sha || '') ||
      commit.parents[0].sha !== tested.parents[0].sha ||
      !/^[0-9a-f]{40}$/.test(commit.commit?.tree?.sha || '') ||
      commit.commit.tree.sha !== tested.commit?.tree?.sha) {
    throw new Error('Recovery requires a single-commit PR with matching tested/merged parents and trees.');
  }
  // Multi-commit rebase merges make the last commit's parent part of the PR.
  // Restrict recovery to one tested commit with an unchanged parent and tree;
  // this proves the immutable base without trusting GitHub's mutable base.sha.
  const bound = {...pr, base: {...pr.base, sha: commit.parents[0].sha}};
  const identity = JSON.stringify([bound.number, head, pr.head.repo.full_name,
    pr.base.repo.full_name, pr.base.ref, pr.merge_commit_sha, bound.base.sha, commit.commit.tree.sha]);
  return {pr: bound, identity};
}

async function prepareMergedRecovery({github, context, core, number, head,
  root = process.env.GITHUB_WORKSPACE}) {
  const {pr, identity} = await mergedRecoveryBinding(github, context, number, head);
  if ((await currentStatus(github, context, head))?.state !== 'pending') return;
  const workflows = await evidence(github, context, pr, root);
  core.info(JSON.stringify(workflows));
  if (!workflows.passed) throw new Error('Required merged-head workflows have not passed.');
  core.setOutput('pr_number', String(pr.number));
  core.setOutput('head_sha', head);
  core.setOutput('snapshot', identity);
  core.setOutput('ready', 'true');
}

async function finalizeMergedRecovery({github, context, core, number, head, before,
  pollPassed, root = process.env.GITHUB_WORKSPACE}) {
  if (!number || !head || !pollPassed) return;
  const {pr, identity} = await mergedRecoveryBinding(github, context, number, head);
  if (identity !== before) throw new Error('Merged recovery identity changed.');
  if ((await currentStatus(github, context, head))?.state !== 'pending') return;
  const [workflows, checks] = await Promise.all([
    evidence(github, context, pr, root),
    verifyCheckRuns({github, ...context.repo, sha: head}),
  ]);
  core.info(JSON.stringify({workflows, checks}));
  if (!workflows.passed || checks.state !== 'success') {
    throw new Error('Required merged-head CI evidence is not green.');
  }
  const after = await mergedRecoveryBinding(github, context, number, head);
  if (after.identity !== before) throw new Error('Merged recovery identity changed before publication.');
  if ((await currentStatus(github, context, head))?.state !== 'pending') return;
  const written = await publish(github, context, after.pr, 'success',
    successDescription(after.pr, workflows.baseSha), false, 'pending', null, false);
  if (!written) return;
  // Publication is not atomic with GitHub's CI state. Recheck and revoke only
  // the exact status this attempt wrote, preserving another writer's update.
  try {
    const final = await mergedRecoveryBinding(github, context, number, head);
    if (final.identity !== before) throw new Error('Merged identity changed during publication.');
    const results = await Promise.all([
      evidence(github, context, final.pr, root),
      verifyCheckRuns({github, ...context.repo, sha: head}),
    ]);
    if (results.some(result => result.state !== 'success')) {
      throw new Error('CI evidence changed during merged recovery publication.');
    }
  } catch (error) {
    await publish(github, context, after.pr, 'failure',
      'Merged recovery evidence changed; inspect CI Check.', false, 'success', written.id, false);
    throw error;
  }
}

module.exports = {prepareMergedRecovery, finalizeMergedRecovery, recoveryCandidates, revokeStaleGreens, snapshot, resolve, freshAfter, prepare, finalize};
