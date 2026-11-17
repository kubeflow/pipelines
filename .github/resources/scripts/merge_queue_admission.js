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

const {workflowAttempts, publishedAttempts} = require('./ci_workflow_attempts');
const {recoverPRGate} = require('./merge_queue_pr_gate_recovery');
const {invalidRunMetadata} = require('./ci_expected_workflows');
const {QUEUE_BRANCHES, loadPolicy, enabledBranches, queueEnabled, requireQueueRule} = require('./merge_queue_policy');
const {checkDependencyConflict} = require('./queue_dependency_conflicts');
const SHA = /^[0-9a-f]{40}$/;
const QUEUE = `query($owner:String!, $repo:String!, $branch:String!, $ref:String!) {
  repository(owner:$owner, name:$repo) {
    nameWithOwner
    baseRef: ref(qualifiedName:$ref) { target { oid } }
    mergeQueue(branch:$branch) {
      entries(first:100) { totalCount pageInfo { hasNextPage } nodes {
        id position headCommit { oid } baseCommit { oid }
        pullRequest { number headRefOid baseRefName }
      } }
    }
  }
}`;

function policyReason(pr, repository, branch) {
  if (!pr || pr.base?.repo?.full_name !== repository || pr.base.ref !== branch ||
      !QUEUE_BRANCHES.includes(branch) || !SHA.test(pr.head?.sha || '') ||
      typeof pr.node_id !== 'string' || !pr.node_id || !Array.isArray(pr.labels) ||
      pr.labels.some(label => typeof label?.name !== 'string' || !label.name) ||
      typeof pr.draft !== 'boolean' || typeof pr.merged !== 'boolean' ||
      typeof pr.user?.login !== 'string' || !pr.user.login || !['User', 'Bot'].includes(pr.user.type)) {
    return 'Incomplete or mismatched PR identity';
  }
  if (pr.state !== 'open' || pr.merged === true || pr.draft) return 'PR is closed or draft';
  const labels = new Set(pr.labels.map(label => label.name));
  if ([...labels].some(label => label === 'do-not-merge' || label.startsWith('do-not-merge/') ||
      ['needs-rebase', 'needs-ok-to-test'].includes(label))) return 'PR has a merge hold';
  // Match the verified Dependabot identity on every explicitly enabled queue.
  if (pr.user?.login === 'dependabot[bot]' && pr.user.type === 'Bot') return null;
  if (!labels.has('lgtm') || !labels.has('approved')) return 'Prow lgtm and approved are required';
  return null;
}

function snapshot(pr) {
  return JSON.stringify([pr.node_id, pr.number, pr.state, pr.draft, pr.merged,
    pr.head?.sha, pr.base?.ref, pr.base?.repo?.full_name, pr.user?.login, pr.user?.type,
    pr.labels?.map(label => label.name).sort()]);
}

async function readPR(github, context, number) {
  const {data} = await github.rest.pulls.get({...context.repo, pull_number: number});
  if (data.number !== number) throw new Error('PR response number mismatch');
  return data;
}

async function readQueue(github, context, branch) {
  return (await readQueueSnapshot(github, context, branch)).entries;
}

async function readQueueSnapshot(github, context, branch) {
  const response = await github.graphql(QUEUE, {...context.repo, branch, ref: `refs/heads/${branch}`});
  return {base: response?.repository?.baseRef?.target?.oid, entries: queueEntries(response, context, branch)};
}

function queueEntries(response, context, branch) {
  const repo = response?.repository;
  const entries = repo?.mergeQueue?.entries;
  if (response?.errors !== undefined || repo?.nameWithOwner !== `${context.repo.owner}/${context.repo.repo}` ||
      !entries || !Array.isArray(entries.nodes) || !Number.isSafeInteger(entries.totalCount) ||
      entries.totalCount !== entries.nodes.length || entries.totalCount > 100 ||
      entries.pageInfo?.hasNextPage !== false) throw new Error('Incomplete merge queue membership');
  const firstPosition = entries.nodes[0]?.position ?? 1;
  if (![0, 1].includes(firstPosition)) throw new Error('Incomplete merge queue positions');
  const ids = new Set(), numbers = new Set();
  for (const [index, entry] of entries.nodes.entries()) {
    if (typeof entry?.id !== 'string' || !entry.id || ids.has(entry.id) ||
        entry.position !== index + firstPosition || !Number.isSafeInteger(entry.pullRequest?.number) ||
        entry.pullRequest.number <= 0 || numbers.has(entry.pullRequest.number) ||
        entry.pullRequest.baseRefName !== branch || !SHA.test(entry.pullRequest.headRefOid || '') ||
        (entry.headCommit !== null && !SHA.test(entry.headCommit?.oid || '')) ||
        (entry.baseCommit !== null && !SHA.test(entry.baseCommit?.oid || ''))) {
      throw new Error('Invalid or reordered merge queue membership');
    }
    ids.add(entry.id);
    numbers.add(entry.pullRequest.number);
  }
  return entries.nodes;
}

async function readQueuePRs(github, context, entries) {
  if (!entries.length) return [];
  // One fresh snapshot per validation boundary, not one REST request per PR.
  // Numbers were validated by readQueue before interpolation into field aliases.
  const fields = entries.map(entry => `pr${entry.pullRequest.number}: pullRequest(number:${entry.pullRequest.number}) {
    id number state isDraft merged headRefOid baseRefName
    baseRepository { nameWithOwner }
    author { __typename login }
    labels(first:100) { totalCount pageInfo { hasNextPage } nodes { name } }
  }`);
  const response = await github.graphql(`query QueuePolicies($owner:String!, $repo:String!) {
    repository(owner:$owner, name:$repo) { nameWithOwner ${fields.join('\n')} }
  }`, context.repo);
  const repo = response?.repository;
  if (response?.errors !== undefined || repo?.nameWithOwner !== `${context.repo.owner}/${context.repo.repo}`) {
    throw new Error('Incomplete queue policy snapshot');
  }
  return entries.map(entry => queuePR(repo[`pr${entry.pullRequest.number}`], entry));
}

function queuePR(pr, entry) {
  const labels = pr?.labels;
  if (pr?.number !== entry.pullRequest.number ||
      !labels || !Array.isArray(labels.nodes) || labels.nodes.length > 100 ||
      labels.totalCount !== labels.nodes.length || labels.pageInfo?.hasNextPage !== false ||
      !['OPEN', 'CLOSED', 'MERGED'].includes(pr.state)) {
    throw new Error('Incomplete queue PR policy');
  }
  const type = pr.author?.__typename;
  const login = type === 'Bot' && pr.author.login && !pr.author.login.endsWith('[bot]') ?
    `${pr.author.login}[bot]` : pr.author?.login;
  return {node_id: pr.id, number: pr.number, state: pr.state.toLowerCase(),
    draft: pr.isDraft, merged: pr.merged, head: {sha: pr.headRefOid},
    base: {ref: pr.baseRefName, repo: {full_name: pr.baseRepository?.nameWithOwner}},
    user: {login, type}, labels: labels.nodes};
}

async function inspectQueue({github, context, group, root}) {
  if (!await queueEnabled({github, context, branch: group.baseRef, root})) return null;
  const entries = await readQueue(github, context, group.baseRef);
  const policies = await readQueuePRs(github, context, entries);
  return queueFingerprint(entries, policies, context, group);
}

function queueFingerprint(entries, policies, context, group) {
  const candidateIndex = entries.findIndex(entry => entry.headCommit?.oid === group.head);
  if (candidateIndex === -1) {
    throw new Error('Candidate is not in the current merge queue');
  }
  const members = [];
  // GitHub exposes entries, not an authoritative group-members connection.
  // Conservatively require every queued PR to remain eligible. A held later
  // entry may temporarily block an earlier candidate until it is dequeued.
  for (const [index, entry] of entries.entries()) {
    const pr = policies[index];
    const reason = policyReason(pr, `${context.repo.owner}/${context.repo.repo}`, group.baseRef);
    if (reason || pr.head.sha !== entry.pullRequest.headRefOid) {
      throw new Error(`Queue PR #${entry.pullRequest.number}: ${reason || 'head changed'}`);
    }
    members.push([pr.node_id, pr.number, pr.head.sha, pr.base.ref, pr.base.repo.full_name]);
  }
  // Later eligible entries and unrelated label edits cannot change this
  // candidate. Fingerprint its queue prefix only, while checking every entry's
  // current approval/hold policy on each read above.
  return JSON.stringify([entries.slice(0, candidateIndex + 1), members.slice(0, candidateIndex + 1)]);
}

// Read all mutable dispatch identities together at each validation boundary.
// No snapshot is reused between reservations or workflow dispatches.
async function inspectDispatchSnapshot({github, context, group, root}) {
  const configured = loadPolicy(root).enabled_branches;
  if (!QUEUE_BRANCHES.includes(group.baseRef) ||
      (!configured.includes(group.baseRef) &&
       !(configured.includes('~DEFAULT_BRANCH') && ['master', 'main'].includes(group.baseRef)))) return null;
  const response = await github.graphql(`query DispatchSnapshot(
    $owner:String!, $repo:String!, $branch:String!, $candidate:String!, $base:String!) {
    repository(owner:$owner, name:$repo) {
      nameWithOwner defaultBranchRef { name }
      candidateRef: ref(qualifiedName:$candidate) { target { __typename ... on Commit { oid } } }
      baseRef: ref(qualifiedName:$base) { target { __typename ... on Commit { oid } } }
      mergeQueue(branch:$branch) { entries(first:100) {
        totalCount pageInfo { hasNextPage } nodes {
          id position headCommit { oid } baseCommit { oid }
          pullRequest {
            id number state isDraft merged headRefOid baseRefName
            baseRepository { nameWithOwner } author { __typename login }
            labels(first:100) { totalCount pageInfo { hasNextPage } nodes { name } }
          }
        }
      } }
    }
  }`, {...context.repo, branch: group.baseRef,
    candidate: `refs/heads/${group.branch}`, base: `refs/heads/${group.baseRef}`});
  const repo = response?.repository;
  if (response?.errors !== undefined || repo?.nameWithOwner !== `${context.repo.owner}/${context.repo.repo}`) {
    throw new Error('Incomplete queue dispatch snapshot');
  }
  if (configured.includes('~DEFAULT_BRANCH')) {
    if (!['master', 'main'].includes(repo.defaultBranchRef?.name)) {
      throw new Error('Cannot resolve merge queue default branch to trusted master or main.');
    }
    if (!configured.includes(group.baseRef) && repo.defaultBranchRef.name !== group.baseRef) return null;
  }
  await requireQueueRule({github, context, branch: group.baseRef});
  for (const ref of [repo.candidateRef, repo.baseRef]) {
    if (ref?.target?.__typename !== 'Commit' || !SHA.test(ref.target.oid || '')) {
      throw new Error('Incomplete queue dispatch ref identity');
    }
  }
  const rawEntries = queueEntries(response, context, group.baseRef);
  const policies = rawEntries.map(entry => queuePR(entry.pullRequest, entry));
  // Keep the fingerprint identical to inspectQueue: policy fields are validated
  // but unrelated label edits and later eligible entries do not change it.
  const entries = rawEntries.map(entry => ({id: entry.id, position: entry.position,
    headCommit: entry.headCommit, baseCommit: entry.baseCommit,
    pullRequest: {number: entry.pullRequest.number, headRefOid: entry.pullRequest.headRefOid,
      baseRefName: entry.pullRequest.baseRefName}}));
  return {head: repo.candidateRef.target.oid, base: repo.baseRef.target.oid,
    policy: queueFingerprint(entries, policies, context, group)};
}

const RETRY_COOLDOWN_MS = 15 * 60 * 1000;

async function latestQueueEvent(github, context, pr) {
  const response = await github.graphql(`query RetryRemoval($owner:String!, $repo:String!, $number:Int!) {
    repository(owner:$owner, name:$repo) { nameWithOwner pullRequest(number:$number) {
      number headRefOid baseRefName
      timelineItems(last:1, itemTypes:[ADDED_TO_MERGE_QUEUE_EVENT, REMOVED_FROM_MERGE_QUEUE_EVENT]) {
        nodes { __typename ... on RemovedFromMergeQueueEvent {
          id reason createdAt beforeCommit { oid }
        } }
      }
    } }
  }`, {...context.repo, number:pr.number});
  const current = response?.repository?.pullRequest;
  const events = current?.timelineItems?.nodes;
  if (response?.errors !== undefined ||
      response?.repository?.nameWithOwner !== `${context.repo.owner}/${context.repo.repo}` ||
      current?.number !== pr.number || current.headRefOid !== pr.head.sha ||
      current.baseRefName !== pr.base.ref || !Array.isArray(events) || events.length > 1) {
    throw new Error('Incomplete queue retry removal evidence');
  }
  return events[0];
}

async function retryRemoval(github, context, pr, consumed, now) {
  const admittedAt = Date.parse(consumed.created_at);
  if (!Number.isFinite(admittedAt) || admittedAt > now()) return null;
  const event = await latestQueueEvent(github, context, pr);
  if (!event || event.__typename !== 'RemovedFromMergeQueueEvent' || event.reason !== 'failed_checks') return null;
  const removedAt = Date.parse(event.createdAt);
  if (typeof event.id !== 'string' || !event.id || !SHA.test(event.beforeCommit?.oid || '') ||
      !Number.isFinite(removedAt) || removedAt < admittedAt || removedAt > now()) return null;
  if (now() - removedAt < RETRY_COOLDOWN_MS) return null;
  return event.id;
}

function readStatuses(github, context, pr) {
  return github.paginate(github.rest.repos.listCommitStatusesForRef, {
    ...context.repo, ref: pr.head.sha, per_page: 100,
  });
}

async function admissionPassed(github, context, pr, {now = () => Date.now(), reservation, afterEnqueue = false, onFailure, statuses} = {}) {
  statuses ??= await readStatuses(github, context, pr);
  const status = statuses.find(status => status.context === 'ci-passed');
  if (status?.state === 'failure' && onFailure) await onFailure({github, context, pr});
  const prefix = `Queue admission ${pr.base.ref} CI passed; base policy `;
  if (!(QUEUE_BRANCHES.includes(pr.base.ref) && status?.state === 'success' &&
    Number.isSafeInteger(status.id) && status.id > 0 && (status.description || '').startsWith(prefix) &&
    /^[0-9a-f]{64}\.$/.test((status.description || '').slice(prefix.length)))) return null;
  const runs = await github.paginate(github.rest.actions.listWorkflowRunsForRepo, {
    ...context.repo, event: 'pull_request', head_sha: pr.head.sha, per_page: 100,
  });
  if (!Array.isArray(runs) || !runs.length || runs.length >= 1000) {
    throw new Error('Incomplete admission workflow history');
  }
  const ids = new Set();
  for (const run of runs) {
    if (invalidRunMetadata(run) || run.event !== 'pull_request' ||
        run.head_sha !== pr.head.sha || ids.has(run.id)) {
      throw new Error('Invalid admission workflow attempt');
    }
    ids.add(run.id);
  }
  const fingerprint = workflowAttempts(runs, pr.head.sha);
  if (publishedAttempts(status, `${context.repo.owner}/${context.repo.repo}`) !== fingerprint) return null;
  const consumed = statuses.find(item => item.context === 'queue-admission-attempt');
  if (consumed?.description !== `Admission ${pr.base.ref}; CI attempts ${fingerprint}.`) {
    return {fingerprint};
  }
  const description = `Retry ${pr.base.ref}; CI attempts ${fingerprint}.`;
  const retry = statuses.find(item => item.context === 'queue-admission-retry' && item.description === description);
  const compensationEvent = item => {
    const prefix = `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/`;
    const marker = `#queue-retry-compensation=${consumed.id}:${fingerprint}:`;
    const url = item?.target_url;
    if (item?.context !== 'queue-admission-retry' || item.state !== 'error' ||
        item.creator?.type !== 'Bot' || item.creator.login !== 'github-actions[bot]' ||
        !Number.isSafeInteger(item.id) || item.id <= 0 ||
        !Number.isSafeInteger(consumed.id) || consumed.id <= 0 || typeof url !== 'string' ||
        !url.startsWith(prefix)) return null;
    const [run, encodedEvent, ...extra] = url.slice(prefix.length).split(marker);
    if (!/^[1-9][0-9]*$/.test(run) || !encodedEvent || extra.length) return null;
    try {
      const event = decodeURIComponent(encodedEvent);
      return encodeURIComponent(event) === encodedEvent ? event : null;
    } catch { return null; }
  };
  const stillCompensated = async item => {
    const expected = compensationEvent(item);
    if (!expected) return false;
    const latest = await latestQueueEvent(github, context, pr);
    return latest?.__typename === 'RemovedFromMergeQueueEvent' && latest.id === expected;
  };
  if (retry) {
    // Only the unchanged, confirmed compensating removal releases a retry.
    // A later manual queue event invalidates this receipt even on the same head.
    if (await stillCompensated(retry)) {
      return {fingerprint, removal: compensationEvent(retry), compensation: retry.id};
    }
    // Only this invocation may finish its pending reservation. Lost responses
    // and failed cleanup keep the retry consumed.
    if (!reservation || retry.id !== reservation.id || retry.state !== 'pending' ||
        reservation.fingerprint !== fingerprint) return null;
    if (!afterEnqueue) {
      if (reservation.admission.compensation) {
        const receipt = statuses.find(item => item.id === reservation.admission.compensation &&
          item.description === description);
        if (!await stillCompensated(receipt)) return null;
      } else if (await retryRemoval(github, context, pr, consumed, now) !== reservation.admission.removal) return null;
    }
    return reservation.admission;
  }
  const removal = await retryRemoval(github, context, pr, consumed, now);
  return removal ? {fingerprint, removal} : null;
}

async function invalidateQueue(github, context, entries) {
  const heads = new Set(entries.map(entry => entry.headCommit?.oid).filter(Boolean));
  for (const sha of heads) {
    await github.rest.repos.createCommitStatus({...context.repo, sha,
      context: 'ci-passed', state: 'failure',
      description: 'Queue approval or hold policy changed; waiting for a rebuilt candidate.'});
  }
}

async function dequeue(github, pr) {
  await github.graphql(`mutation($id:ID!) {
    dequeuePullRequest(input:{id:$id}) { clientMutationId }
  }`, {id: pr.node_id});
}

async function reconcilePR({github, context, number, branch, root, core, now = () => Date.now(), onFailure,
  conflictCheck = checkDependencyConflict, cache = new Map()}) {
  if (!await queueEnabled({github, context, branch, root})) return;
  const initialQueue = await readQueueSnapshot(github, context, branch);
  let entries = initialQueue.entries;
  const entry = entries.find(entry => entry.pullRequest.number === number);
  let pr;
  let enqueueAttempted = false;
  let recordAcknowledgedAdmission;
  try {
    pr = await readPR(github, context, number);
    const reason = policyReason(pr, `${context.repo.owner}/${context.repo.repo}`, branch);
    if (entry && (reason || entry.pullRequest.headRefOid !== pr.head.sha)) {
      await invalidateQueue(github, context, entries);
      await dequeue(github, pr);
      return;
    }
    if (entry || reason) return;
    const recoverFailure = onFailure || (async ({pr: failedPR}) => {
      const beforeRecovery = snapshot(failedPR);
      await recoverPRGate({github, context, pr: failedPR, now,
        core: core || {info: () => {}}, revalidate: async () => {
          const current = await readPR(github, context, failedPR.number);
          if (snapshot(current) !== beforeRecovery ||
              policyReason(current, `${context.repo.owner}/${context.repo.repo}`, branch)) return false;
          const statuses = await readStatuses(github, context, failedPR);
          if (!Array.isArray(statuses) || statuses.length >= 1000) {
            throw new Error('Incomplete PR Gate recovery status evidence');
          }
          return statuses.find(status => status.context === 'ci-passed')?.state === 'failure' &&
            await queueEnabled({github, context, branch, root});
        }});
    });
    const statuses = await readStatuses(github, context, pr);
    const admission = await admissionPassed(github, context, pr, {now, statuses, onFailure: recoverFailure});
    if (!admission) return;
    if (!SHA.test(initialQueue.base || '')) throw new Error('Missing live base for queue admission');
    const before = snapshot(pr);
    if (entries.length && await conflictCheck({github, context, pr, entries,
      base: initialQueue.base, core, cache, statuses})) return;
    pr = await readPR(github, context, number);
    if (snapshot(pr) !== before || JSON.stringify(await admissionPassed(github, context, pr, {now})) !== JSON.stringify(admission)) return;
    if (!await queueEnabled({github, context, branch, root})) return;
    // Manual queue changes can race even an empty queue. Compare the live base,
    // ordered membership and candidate SHAs before reserving a retry or enqueuing.
    const currentQueue = await readQueueSnapshot(github, context, branch);
    if (JSON.stringify(currentQueue) !== JSON.stringify(initialQueue)) return;
    let reservation;
    if (admission.removal) {
      const {data} = await github.rest.repos.createCommitStatus({...context.repo, sha:pr.head.sha,
        context:'queue-admission-retry', state:'pending',
        description:`Retry ${branch}; CI attempts ${admission.fingerprint}.`});
      if (!Number.isSafeInteger(data?.id) || data.id <= 0) throw new Error('Invalid queue retry reservation response');
      reservation = {id:data.id, fingerprint:admission.fingerprint, admission};
      pr = await readPR(github, context, number);
      if (snapshot(pr) !== before || JSON.stringify(await admissionPassed(github, context, pr, {now, reservation})) !== JSON.stringify(admission) ||
          !await queueEnabled({github, context, branch, root})) return;
    }
    // No direct-merge or auto-merge fallback. GitHub enforces its other required
    // checks, including DCO, and atomically rejects a changed PR head.
    // GITHUB_TOKEN admission does not reliably trigger merge_group Actions runs;
    // merge_queue_dispatch supplies missing CI via the workflow_dispatch exception.
    enqueueAttempted = true;
    await github.graphql(`mutation($id:ID!, $head:GitObjectID!) {
      enqueuePullRequest(input:{pullRequestId:$id,expectedHeadOid:$head,jump:false}) {
        mergeQueueEntry { id }
      }
    }`, {id: pr.node_id, head: pr.head.sha});
    let admissionRecorded = false;
    recordAcknowledgedAdmission = async () => {
      if (admissionRecorded) return;
      await github.rest.repos.createCommitStatus({...context.repo, sha: pr.head.sha,
        context: 'queue-admission-attempt', state: 'pending',
        description: `Admission ${branch}; CI attempts ${admission.fingerprint}.`});
      admissionRecorded = true;
    };
    const after = await readPR(github, context, number);
    if (after.merged === true) return;
    const afterQueue = await readQueueSnapshot(github, context, branch);
    const admittedIndex = afterQueue.entries.findIndex(item => item.pullRequest.number === number);
    if (admittedIndex < 0) {
      // GitHub may remove a candidate before the first snapshot. That
      // acknowledged attempt still consumes its budget without compensation.
      await recordAcknowledgedAdmission();
      return;
    }
    const predecessors = {base: afterQueue.base, entries: afterQueue.entries.slice(0, admittedIndex)};
    if (JSON.stringify(predecessors) !== JSON.stringify(initialQueue)) {
      // A manual enqueue or base move raced our final pre-enqueue snapshot.
      // Remove this admission before recording consumption; do not invalidate
      // unrelated predecessors or accept their unprobed dependency changes.
      await dequeue(github, after);
      const removed = await readQueueSnapshot(github, context, branch);
      if (removed.entries.some(item => item.pullRequest.number === number)) {
        throw new Error('Compensating dequeue not confirmed');
      }
      if (reservation) {
        const removal = await latestQueueEvent(github, context, after);
        if (removal?.__typename !== 'RemovedFromMergeQueueEvent' || typeof removal.id !== 'string' || !removal.id || removal.id === admission.removal) {
          throw new Error('Compensating removal event not confirmed');
        }
        const consumed = statuses.find(item => item.context === 'queue-admission-attempt');
        if (!Number.isSafeInteger(consumed?.id) || consumed.id <= 0) throw new Error('Missing consumed admission for compensation');
        await github.rest.repos.createCommitStatus({...context.repo, sha:pr.head.sha,
          context:'queue-admission-retry', state:'error',
          description:`Retry ${branch}; CI attempts ${admission.fingerprint}.`,
          target_url:`https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}#queue-retry-compensation=${consumed.id}:${admission.fingerprint}:${encodeURIComponent(removal.id)}`});
      }
      return;
    }
    if (snapshot(after) !== before || JSON.stringify(await admissionPassed(github, context, after, {now, reservation, afterEnqueue:true})) !== JSON.stringify(admission) ||
        !await queueEnabled({github, context, branch, root})) {
      entries = await readQueue(github, context, branch);
      await invalidateQueue(github, context, entries);
      await dequeue(github, after);
    } else {
      // Each CI fingerprint gets one initial admission and one delayed retry.
      await recordAcknowledgedAdmission();
    }
  } catch (error) {
    // An enqueue may have succeeded even if its response was lost. Refresh the
    // queue and attempt removal independently of status/API failures.
    const errors = [error];
    // An acknowledged enqueue followed by an unreadable snapshot must not
    // silently regain its initial-admission budget either.
    if (recordAcknowledgedAdmission) {
      try { await recordAcknowledgedAdmission(); }
      catch (statusError) { errors.push(statusError); }
    }
    let removalRequired = Boolean(entry);
    if (enqueueAttempted) {
      try {
        entries = await readQueue(github, context, branch);
        removalRequired = entries.some(current => current.pullRequest.number === number);
      } catch (refreshError) {
        errors.push(refreshError);
        removalRequired = true;
      }
    }
    // A rejected enqueue with confirmed absence must not revoke unrelated
    // candidates. An unreadable response still requires conservative cleanup.
    if (removalRequired) {
      try { await invalidateQueue(github, context, entries); }
      catch (statusError) { errors.push(statusError); }
      if (pr) {
        try { await dequeue(github, pr); }
        catch (dequeueError) { errors.push(dequeueError); }
      }
    }
    throw new Error(errors.map(failure => failure.message).join('; '));
  }
}

async function eventTargets(github, context) {
  if (context.eventName === 'schedule') return null;
  if (context.eventName === 'workflow_dispatch') {
    const branch = context.payload.repository?.default_branch;
    const number = Number(context.payload.inputs?.pr_number);
    const head = context.payload.inputs?.head_sha;
    if (!['main', 'master'].includes(branch) || context.ref !== `refs/heads/${branch}` ||
        !Number.isSafeInteger(number) || number <= 0 || !SHA.test(head || '')) {
      throw new Error('Invalid trusted queue admission wakeup');
    }
    const pr = await readPR(github, context, number);
    return pr.head?.sha === head ? [number] : [];
  }
  if (context.eventName === 'pull_request_target') {
    const number = context.payload.pull_request?.number;
    if (!Number.isSafeInteger(number) || number <= 0) throw new Error('Invalid PR event number');
    return [number];
  }
  if (context.eventName !== 'workflow_run') throw new Error('Unsupported queue admission event');
  const run = context.payload.workflow_run;
  if (run?.event !== 'pull_request') return [];
  if (!SHA.test(run.head_sha || '') || typeof run.head_branch !== 'string' || !run.head_branch ||
      typeof run.head_repository?.owner?.login !== 'string' || !run.head_repository.owner.login ||
      typeof run.head_repository.full_name !== 'string' ||
      !run.head_repository.full_name.startsWith(`${run.head_repository.owner.login}/`)) {
    throw new Error('Invalid workflow head identity for queue admission');
  }
  const prs = await github.paginate(github.rest.pulls.list, {
    ...context.repo, state: 'open',
    head: `${run.head_repository.owner.login}:${run.head_branch}`, per_page: 100,
  });
  if (!Array.isArray(prs)) throw new Error('Invalid workflow PR discovery response');
  return prs.filter(pr => pr.head?.sha === run.head_sha &&
    pr.head.repo?.full_name === run.head_repository.full_name).map(pr => pr.number);
}

async function sweep({github, context, root, core, queueOnly = false, now, onFailure, conflictCheck}) {
  const branches = await enabledBranches({github, context, root});
  if (!branches.length) return;
  const targets = queueOnly ? [] : await eventTargets(github, context);
  const failures = [];
  const cache = new Map();
  for (const branch of branches) {
    // A broken queue must not prevent other enabled branches from progressing.
    try {
      if (!await queueEnabled({github, context, branch, root})) continue;
      const entries = await readQueue(github, context, branch);
      const listed = await github.paginate(github.rest.pulls.list, {
        ...context.repo, base: branch, state: 'open', per_page: 100,
      });
      if (!Array.isArray(listed)) throw new Error('Invalid ready PR discovery response');
      // Concurrency coalesces pending wakeups. Every surviving writer catches up
      // eligible green PRs, including the five-minute queue recovery. The label
      // is only a cheap filter; reconcilePR rechecks authoritative fresh evidence.
      const ready = listed.filter(pr => pr.labels?.some(label => label.name === 'ci-passed') &&
        !policyReason({...pr, merged: false}, `${context.repo.owner}/${context.repo.repo}`, branch));
      const affected = targets ?? listed.map(pr => pr.number);
      const numbers = new Set([...affected, ...ready.map(pr => pr.number),
        ...entries.map(entry => entry.pullRequest.number)]);
      if (numbers.size > 256 || [...numbers].some(number => !Number.isSafeInteger(number) || number <= 0)) {
        throw new Error('Queue admission sweep exceeds bounds or has invalid PR identities');
      }
      let policies;
      try {
        policies = await readQueuePRs(github, context, entries);
      } catch (error) {
        await invalidateQueue(github, context, entries);
        throw error;
      }
      const healthy = new Set(entries.filter((entry, index) =>
        !policyReason(policies[index], `${context.repo.owner}/${context.repo.repo}`, branch) &&
        policies[index].head.sha === entry.pullRequest.headRefOid).map(entry => entry.pullRequest.number));
      for (const number of numbers) {
        // Healthy queued members cannot be newly admitted. Invalid members retain
        // the existing fresh read and fail-closed invalidation/dequeue path.
        if (healthy.has(number)) continue;
        try {
          await reconcilePR({github, context, number, branch, root, core, now, onFailure, conflictCheck, cache});
        } catch (error) {
          failures.push(`#${number}: ${error.message}`);
        }
      }
    } catch (error) {
      failures.push(`${branch}: ${error.message}`);
    }
  }
  if (failures.length) throw new Error(failures.join('; '));
  core.info('Merge queue admission sweep complete.');
}

module.exports = {policyReason, snapshot, readQueue, inspectQueue, inspectDispatchSnapshot, reconcilePR, sweep};
