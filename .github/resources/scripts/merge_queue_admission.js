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

const {loadPolicy, queueEnabled} = require('./merge_queue_policy');
const SHA = /^[0-9a-f]{40}$/;
const QUEUE = `query($owner:String!, $repo:String!, $branch:String!) {
  repository(owner:$owner, name:$repo) { nameWithOwner mergeQueue(branch:$branch) {
    entries(first:100) { totalCount pageInfo { hasNextPage } nodes {
      id position headCommit { oid } baseCommit { oid }
      pullRequest { number headRefOid baseRefName }
    } }
  } }
}`;

function policyReason(pr, repository, branch) {
  if (!pr || pr.base?.repo?.full_name !== repository || pr.base.ref !== branch ||
      !['master', 'main'].includes(branch) || !SHA.test(pr.head?.sha || '') ||
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
  // Match the existing development-branch Dependabot exception, not arbitrary bots.
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
  const response = await github.graphql(QUEUE, {...context.repo, branch});
  const repo = response?.repository;
  const entries = repo?.mergeQueue?.entries;
  if (repo?.nameWithOwner !== `${context.repo.owner}/${context.repo.repo}` ||
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

async function inspectQueue({github, context, group, root}) {
  if (!await queueEnabled({github, context, branch: group.baseRef, root})) return null;
  const entries = await readQueue(github, context, group.baseRef);
  const candidateIndex = entries.findIndex(entry => entry.headCommit?.oid === group.head);
  if (candidateIndex === -1) {
    throw new Error('Candidate is not in the current merge queue');
  }
  const members = [];
  // GitHub exposes entries, not an authoritative group-members connection.
  // Conservatively require every queued PR to remain eligible. A held later
  // entry may temporarily block an earlier candidate until it is dequeued.
  for (const entry of entries) {
    const pr = await readPR(github, context, entry.pullRequest.number);
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

async function admissionPassed(github, context, pr) {
  const statuses = await github.paginate(github.rest.repos.listCommitStatusesForRef, {
    ...context.repo, ref: pr.head.sha, per_page: 100,
  });
  const status = statuses.find(status => status.context === 'ci-passed');
  return ['master', 'main'].includes(pr.base.ref) && status?.state === 'success' &&
    new RegExp(`^Queue admission ${pr.base.ref} CI passed; base policy [0-9a-f]{64}\\.$`).test(status.description || '');
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
    dequeuePullRequest(input:{pullRequestId:$id}) { clientMutationId }
  }`, {id: pr.node_id});
}

async function reconcilePR({github, mutationGithub = github, context, number, branch, root}) {
  if (!await queueEnabled({github, context, branch, root})) return;
  let entries = await readQueue(github, context, branch);
  const entry = entries.find(entry => entry.pullRequest.number === number);
  let pr;
  let enqueueAttempted = false;
  try {
    pr = await readPR(github, context, number);
    const reason = policyReason(pr, `${context.repo.owner}/${context.repo.repo}`, branch);
    if (entry && (reason || entry.pullRequest.headRefOid !== pr.head.sha)) {
      await invalidateQueue(github, context, entries);
      await dequeue(mutationGithub, pr);
      return;
    }
    if (entry || reason || !await admissionPassed(github, context, pr)) return;
    const before = snapshot(pr);
    pr = await readPR(github, context, number);
    if (snapshot(pr) !== before || !await admissionPassed(github, context, pr)) return;
    if (!await queueEnabled({github, context, branch, root})) return;
    // No direct-merge or auto-merge fallback. GitHub enforces its other required
    // checks, including DCO, and atomically rejects a changed PR head.
    enqueueAttempted = true;
    await mutationGithub.graphql(`mutation($id:ID!, $head:GitObjectID!) {
      enqueuePullRequest(input:{pullRequestId:$id,expectedHeadOid:$head,jump:false}) {
        mergeQueueEntry { id }
      }
    }`, {id: pr.node_id, head: pr.head.sha});
    const after = await readPR(github, context, number);
    if (after.merged === true) return;
    if (snapshot(after) !== before || !await admissionPassed(github, context, after) ||
        !await queueEnabled({github, context, branch, root})) {
      entries = await readQueue(github, context, branch);
      await invalidateQueue(github, context, entries);
      await dequeue(mutationGithub, after);
    }
  } catch (error) {
    // An enqueue may have succeeded even if its response was lost. Refresh the
    // queue and attempt removal independently of status/API failures.
    const errors = [error];
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
        try { await dequeue(mutationGithub, pr); }
        catch (dequeueError) { errors.push(dequeueError); }
      }
    }
    throw new Error(errors.map(failure => failure.message).join('; '));
  }
}

async function eventTargets(github, context) {
  if (context.eventName === 'schedule') return null;
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

async function sweep({github, mutationGithub = github, context, root, core}) {
  const branches = loadPolicy(root).enabled_branches;
  if (!branches.length) return;
  const targets = await eventTargets(github, context);
  const failures = [];
  for (const branch of branches) {
    if (!await queueEnabled({github, context, branch, root})) continue;
    const entries = await readQueue(github, context, branch);
    const affected = targets ?? (await github.paginate(github.rest.pulls.list, {
      ...context.repo, base: branch, state: 'open', per_page: 100,
    })).map(pr => pr.number);
    const numbers = new Set([...affected, ...entries.map(entry => entry.pullRequest.number)]);
    if (numbers.size > 256 || [...numbers].some(number => !Number.isSafeInteger(number) || number <= 0)) {
      throw new Error('Queue admission sweep exceeds bounds or has invalid PR identities');
    }
    for (const number of numbers) {
      try {
        await reconcilePR({github, mutationGithub, context, number, branch, root});
      } catch (error) {
        failures.push(`#${number}: ${error.message}`);
      }
    }
  }
  if (failures.length) throw new Error(failures.join('; '));
  core.info('Merge queue admission sweep complete.');
}

module.exports = {policyReason, snapshot, readQueue, inspectQueue, reconcilePR, sweep};
