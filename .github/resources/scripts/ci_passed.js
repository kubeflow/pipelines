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

const {applicable, verifyExpectedWorkflows, loadBaseInventory} = require('./ci_expected_workflows');

const RELEASE_BRANCH = 'release-2.18';
const QUEUE_BRANCH_PREFIX = `gh-readonly-queue/${RELEASE_BRANCH}/`;
const MASTER_STATUS_CONTEXT = 'ci-passed';
const RELEASE_STATUS_CONTEXT = 'ci-passed-release';
const MAX_WORKFLOW_GUARD_COMMITS = 100;
const MAX_TRUSTED_WORKFLOW_IMPORTS = 8;
const WORKFLOW_TREE_BATCH_SIZE = 40;
const RELEASE_BLOCKED_LABELS = new Set([
  'do-not-merge', 'do-not-merge/hold', 'do-not-merge/invalid-owners-file',
  'do-not-merge/work-in-progress', 'needs-rebase',
]);
// These release workflows include write-capable push or dispatch behavior.
// Queue equivalents execute the needed CI with a read-only token.
const RELEASE_QUEUE_EQUIVALENTS = new Map([
  ['.github/workflows/build-tools-images.yml',
    '.github/workflows/build-tools-images-merge-group.yml'],
  ['.github/workflows/runtime-base-images.yml',
    '.github/workflows/runtime-base-images-merge-group.yml'],
]);
const {verifyCheckRuns} = require('./ci_check_runs');

function eligible(pr) {
  const labels = new Set(pr.labels.map(label => label.name));
  return !labels.has('needs-ok-to-test') && (labels.has('ok-to-test') ||
    pr.user.login === 'dependabot[bot]' ||
    ['MEMBER', 'OWNER', 'COLLABORATOR'].includes(pr.author_association));
}

function admitted(pr) {
  if (!eligible(pr)) return false;
  if (pr.base.ref !== RELEASE_BRANCH) return true;
  const labels = new Set(pr.labels.map(label => label.name));
  if (pr.draft || [...RELEASE_BLOCKED_LABELS].some(label => labels.has(label))) return false;
  // Tide's release Dependabot query does not require human approval labels.
  return pr.user.login === 'dependabot[bot]' ||
    (labels.has('lgtm') && labels.has('approved'));
}

function prStatusContexts(pr) {
  // If a PR reread fails, invalidate both contexts rather than leaving a
  // potentially green release status behind. Unknown-base callers never approve.
  if (!pr.base?.ref) return [RELEASE_STATUS_CONTEXT, MASTER_STATUS_CONTEXT];
  if (pr.base.ref !== RELEASE_BRANCH) return [MASTER_STATUS_CONTEXT];
  // Invalidate the required release status before touching the legacy Tide
  // context if a status write fails partway through publication.
  return [RELEASE_STATUS_CONTEXT, MASTER_STATUS_CONTEXT];
}

function snapshot(pr) {
  return JSON.stringify([pr.number, pr.state, pr.head.sha, pr.base.ref, pr.base.sha, admitted(pr)]);
}

async function readPR(github, context, number) {
  return (await github.rest.pulls.get({...context.repo, pull_number: number})).data;
}

async function currentStatus(github, context, head, name) {
  // Combined-status responses are paginated independently of commit history.
  for await (const response of github.paginate.iterator(github.rest.repos.getCombinedStatusForRef, {
    ...context.repo, ref: head, per_page: 100,
  })) {
    const statuses = response.data.statuses || response.data;
    const status = statuses.find(item => item.context === name);
    if (status) return status;
  }
  return null;
}

async function releaseWorkflowGuard(github, context, pr) {
  if (pr.base.ref !== RELEASE_BRANCH) return {passed: true};
  const repository = `${context.repo.owner}/${context.repo.repo}`.toLowerCase();
  const {login: headOwner} = pr.head.repo?.owner || {};
  const {name: headRepo, full_name: headRepository} = pr.head.repo || {};
  if (!headOwner || !headRepo ||
      headRepository?.toLowerCase() !== `${headOwner}/${headRepo}`.toLowerCase()) {
    return {passed: false, reason: 'Cannot identify the release PR head repository.'};
  }
  // A same-repository branch writer already has permission to edit workflows.
  // Fork PRs cannot introduce write-capable merge_group or queue-ref workflows.
  if (headRepository.toLowerCase() === repository) return {passed: true};
  if (!/^[0-9a-f]{40}$/.test(pr.base.sha || '') ||
      !/^[0-9a-f]{40}$/.test(pr.head.sha || '')) {
    return {passed: false, reason: 'Cannot identify the release PR revisions.'};
  }
  const comparison = (await github.rest.repos.compareCommitsWithBasehead({
    ...context.repo, basehead: `${pr.base.sha}...${headOwner}:${pr.head.sha}`,
    per_page: MAX_WORKFLOW_GUARD_COMMITS,
  })).data;
  const mergeBase = comparison.merge_base_commit?.sha;
  const commits = comparison.commits;
  if (comparison.base_commit?.sha !== pr.base.sha || !/^[0-9a-f]{40}$/.test(mergeBase || '') ||
      !Number.isSafeInteger(comparison.total_commits) ||
      comparison.total_commits !== comparison.ahead_by ||
      comparison.total_commits > MAX_WORKFLOW_GUARD_COMMITS ||
      !Array.isArray(commits) || commits.length !== comparison.total_commits ||
      (commits.length && commits.at(-1)?.sha !== pr.head.sha) ||
      (!commits.length && mergeBase !== pr.head.sha)) {
    return {passed: false, reason: 'Cannot verify the release PR merge base.'};
  }
  const revisions = new Set([mergeBase, pr.head.sha]);
  for (const commit of commits) {
    if (!/^[0-9a-f]{40}$/.test(commit?.sha || '') ||
        !Array.isArray(commit.parents) || commit.parents.length === 0) {
      return {passed: false, reason: 'Cannot verify the release PR commit history.'};
    }
    revisions.add(commit.sha);
    for (const parent of commit.parents) {
      if (!/^[0-9a-f]{40}$/.test(parent?.sha || '')) {
        return {passed: false, reason: 'Cannot verify the release PR commit history.'};
      }
      revisions.add(parent.sha);
    }
  }
  if (revisions.size > 256) {
    return {passed: false, reason: 'Release PR commit history exceeds workflow verification limit.'};
  }
  // Check every fork-only commit, including commits on merged side branches.
  // A final-tree check alone would miss a change followed by a revert.
  const trees = new Map();
  const shas = [...revisions];
  for (let offset = 0; offset < shas.length; offset += WORKFLOW_TREE_BATCH_SIZE) {
    const batch = shas.slice(offset, offset + WORKFLOW_TREE_BATCH_SIZE);
    const fields = batch.map((sha, index) =>
      `r${index}: object(expression: "${sha}:.github/workflows") { ... on Tree { oid } }`);
    const result = await github.graphql(`query ReleaseWorkflowTrees($owner: String!, $repo: String!) {
      repository(owner: $owner, name: $repo) {
        nameWithOwner
        ${fields.join('\n')}
      }
    }`, {owner: headOwner, repo: headRepo});
    const fork = result?.repository;
    if (fork?.nameWithOwner?.toLowerCase() !== headRepository.toLowerCase()) {
      return {passed: false, reason: 'Cannot verify the release workflow trees.'};
    }
    for (const [index, sha] of batch.entries()) {
      const oid = fork[`r${index}`]?.oid;
      if (!/^[0-9a-f]{40}$/.test(oid || '')) {
        return {passed: false, reason: 'Cannot verify the release workflow trees.'};
      }
      trees.set(sha, oid);
    }
  }
  if (trees.get(mergeBase) !== trees.get(pr.head.sha)) {
    return {passed: false,
      reason: 'Fork PR history edits release workflows; a repository writer must land those edits.'};
  }
  const ahead = new Set(commits.map(commit => commit.sha));
  let imports = 0;
  for (const commit of commits) {
    const tree = trees.get(commit.sha);
    if (commit.parents.every(parent => trees.get(parent.sha) === tree)) continue;
    // A fork merge may import the workflow tree from a release ancestor. The
    // matching parent must be the latest shared base at this merge, so an old
    // release commit cannot roll back workflows and later restore them.
    const trustedParents = commit.parents.filter(parent =>
      !ahead.has(parent.sha) && trees.get(parent.sha) === tree);
    if (commit.parents.length < 2 || trustedParents.length === 0 ||
        ++imports > MAX_TRUSTED_WORKFLOW_IMPORTS) {
      return {passed: false,
        reason: 'Fork PR history edits release workflows; a repository writer must land those edits.'};
    }
    const prefix = commit.sha === pr.head.sha ? comparison :
      (await github.rest.repos.compareCommitsWithBasehead({
        ...context.repo, basehead: `${pr.base.sha}...${headOwner}:${commit.sha}`,
        per_page: 1,
      })).data;
    if (prefix.base_commit?.sha !== pr.base.sha ||
        !trustedParents.some(parent => parent.sha === prefix.merge_base_commit?.sha)) {
      return {passed: false, reason: 'Cannot verify the trusted release workflow import.'};
    }
  }
  return {passed: true};
}

function successDescription(pr) {
  // Bind green evidence to the exact checked-in workflow policy. Legacy
  // statuses and statuses from another base must be reconsidered by recovery.
  const policy = [pr.base.ref, pr.base.sha];
  if (pr.base.ref === RELEASE_BRANCH) policy.push('release-workflow-guard-v4');
  const stamp = require('node:crypto').createHash('sha256')
    .update(JSON.stringify(policy)).digest('hex');
  return `Expected CI and all checks passed; base policy ${stamp}.`;
}

async function recoveryCandidates({github, context}) {
  const prs = await github.paginate(github.rest.pulls.list, {
    ...context.repo, state: 'open', per_page: 100,
  });
  const candidates = [];
  for (const pr of prs) {
    if (!eligible(pr) && pr.base.ref !== RELEASE_BRANCH) continue;
    // Revisit green heads when their trusted base policy changes, including
    // statuses published before base-policy stamps were introduced.
    const statuses = [];
    for (const name of prStatusContexts(pr)) {
      statuses.push(await currentStatus(github, context, pr.head.sha, name));
    }
    if (statuses.every(status => status?.state === 'success' &&
        status.description === successDescription(pr)) && admitted(pr)) continue;
    if (!admitted(pr) && statuses.every(status => status?.state !== 'success')) continue;
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
  for (const name of prStatusContexts(pr)) {
    let current;
    try {
      current = await currentStatus(github, context, pr.head.sha, name);
    } catch (error) {
      // A failed read must not prevent invalidating a previously green head.
      if (state === 'success') throw error;
    }
    // Provisional invalidation revokes green without churning existing red
    // or pending statuses. Authoritative evidence may recover red to pending.
    const preserve = provisional && current && current.state !== 'success';
    const boundedDescription = description.slice(0, 140);
    if (!preserve && (current?.state !== state || current.description !== boundedDescription)) {
      await github.rest.repos.createCommitStatus({
        ...context.repo, sha: pr.head.sha, context: name, state,
        description: boundedDescription,
        target_url: `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`,
      });
    }
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
  const guard = await releaseWorkflowGuard(github, context, pr);
  if (!guard.passed) return {passed: false, state: 'failure', reasons: [guard.reason]};
  const [inventory, cutoff] = await Promise.all([
    loadBaseInventory({github, ...context.repo, pullRequest: pr, root}),
    freshAfter(github, context, pr),
  ]);
  const args = {github, ...context.repo, pullRequest: pr, ...inventory, freshAfter: cutoff};
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
    if (sha) {
      const eventPR = context.payload.pull_request;
      const contexts = eventPR ? prStatusContexts(eventPR) :
        [RELEASE_STATUS_CONTEXT, MASTER_STATUS_CONTEXT];
      for (const name of contexts) {
        await github.rest.repos.createCommitStatus({
          ...context.repo, sha, context: name, state: 'failure',
          description: 'Cannot identify the PR for this head; inspect CI Check and retry.',
        });
      }
    }
    throw error;
  }
  if (!pr) return;
  // Set the identity before any further API operation so the always() step
  // can fail closed if inventory loading or evidence retrieval fails.
  core.setOutput('pr_number', String(pr.number));
  core.setOutput('head_sha', pr.head.sha);
  core.setOutput('snapshot', snapshot(pr));
  await publish(github, context, pr, 'pending', 'CI evidence is being revalidated.', true);
  if (pr.state !== 'open' || !admitted(pr)) return;
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
    let reason = 'PR changed or is ineligible; complete current-head CI and retry.';
    if (pr.head.sha === head && pr.state === 'open' && snapshot(pr) === before && admitted(pr)) {
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
      state === 'success' ? successDescription(pr) : reason);
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

async function queueEntries(github, context) {
  const entries = [];
  let cursor = null;
  let configuration;
  do {
    const result = await github.graphql(`query ReleaseQueue($owner: String!, $repo: String!, $cursor: String) {
      repository(owner: $owner, name: $repo) {
        nameWithOwner
        mergeQueue(branch: "release-2.18") {
          configuration { maximumEntriesToBuild maximumEntriesToMerge mergingStrategy }
          entries(first: 100, after: $cursor) {
            nodes { id position headCommit { oid } pullRequest { id number headRefOid } }
            pageInfo { hasNextPage endCursor }
          }
        }
      }
    }`, {...context.repo, cursor});
    const repository = result?.repository;
    if (repository?.nameWithOwner?.toLowerCase() !==
        `${context.repo.owner}/${context.repo.repo}`.toLowerCase()) {
      throw new Error('Cannot verify the release merge queue repository');
    }
    const queue = repository.mergeQueue;
    if (!queue) return {entries: [], configuration: null};
    if (configuration !== undefined &&
        JSON.stringify(configuration) !== JSON.stringify(queue.configuration)) {
      throw new Error('Release merge queue configuration changed during pagination');
    }
    configuration = queue.configuration;
    const page = queue.entries;
    if (!Array.isArray(page?.nodes) || typeof page.pageInfo?.hasNextPage !== 'boolean') {
      throw new Error('Incomplete release merge queue entries');
    }
    entries.push(...page.nodes);
    if (entries.length > 256) throw new Error('Release merge queue exceeds recovery limit');
    cursor = page.pageInfo.hasNextPage ? page.pageInfo.endCursor : null;
    if (page.pageInfo.hasNextPage && !cursor) {
      throw new Error('Release merge queue has an incomplete page cursor');
    }
  } while (cursor);
  return {entries, configuration};
}

function orderedQueueEntries(entries) {
  if (entries.some(entry => !Number.isSafeInteger(entry?.position) || entry.position < 0)) {
    throw new Error('Release merge queue has an invalid entry position');
  }
  const ordered = [...entries].sort((left, right) => left.position - right.position);
  if (ordered.some((entry, index) => index > 0 &&
      entry.position === ordered[index - 1].position)) {
    throw new Error('Release merge queue has duplicate entry positions');
  }
  return ordered;
}

async function queueRecoveryCandidates({github, context}) {
  const {entries} = await queueEntries(github, context);
  const shas = entries.map(entry => entry?.headCommit?.oid).filter(Boolean);
  if (shas.some(sha => !/^[0-9a-f]{40}$/.test(sha)) || new Set(shas).size !== shas.length) {
    throw new Error('Invalid or ambiguous release merge queue heads');
  }
  return shas.map(sha => ({sha}));
}

async function pullRequestQueueEntry(github, context, id) {
  const result = await github.graphql(`query ReleaseQueuedPR($id: ID!) {
    node(id: $id) {
      ... on PullRequest { id mergeQueueEntry { id } }
    }
  }`, {id});
  if (result?.node?.id !== id ||
      !Object.hasOwn(result.node, 'mergeQueueEntry')) {
    throw new Error('Cannot verify the PR merge queue entry');
  }
  return result.node.mergeQueueEntry;
}

async function queueEventCandidates({github, context}) {
  if (context.eventName !== 'pull_request_target') {
    return (await queueHeads({github, context})).map(sha => ({sha}));
  }
  const number = context.payload.pull_request?.number;
  if (!Number.isSafeInteger(number) || number <= 0) return [];
  const {entries} = await queueEntries(github, context);
  const ordered = orderedQueueEntries(entries);
  const matches = ordered.filter(entry => entry?.pullRequest?.number === number);
  if (matches.length > 1) throw new Error('PR has ambiguous release merge queue entries');
  if (matches.length === 0) return [];
  const queued = matches[0];
  const affected = ordered.filter(entry => entry.position >= queued.position);
  const known = [];
  for (const entry of affected) {
    const sha = entry?.headCommit?.oid;
    if (sha == null) continue;
    if (!/^[0-9a-f]{40}$/.test(sha) || known.includes(sha)) {
      throw new Error('Affected release queue heads are invalid or ambiguous');
    }
    known.push(sha);
  }
  const pr = await readPR(github, context, number);
  const isAdmitted = pr.number === number && pr.node_id === queued.pullRequest?.id &&
    pr.state === 'open' && pr.base.ref === RELEASE_BRANCH &&
    pr.head.sha === queued.pullRequest?.headRefOid && admitted(pr);
  if (isAdmitted) {
    // A successor may be queued but not built yet. It has no SHA to refresh
    // and cannot satisfy the required queue status until a build starts.
    return known.map(sha => ({sha}));
  }
  // Dequeue removes the invalid PR, but the old cumulative SHAs may remain
  // green until that mutation finishes. Revoke every known affected SHA first.
  let invalidationError;
  for (const sha of known) {
    try {
      await queueStatus(github, context, sha, 'pending',
        `queue-retired:${context.runId}.${context.runAttempt || process.env.GITHUB_RUN_ATTEMPT || 1}`);
    } catch (error) {
      // The queue mutation may still work when status creation fails. Keep
      // trying to remove the ineligible PR, then report the write failure.
      invalidationError ||= error;
    }
  }
  if (!queued.id || !queued.pullRequest?.id || pr.node_id !== queued.pullRequest.id) {
    throw new Error('Cannot identify the queued PR to remove');
  }
  const current = await readPR(github, context, number);
  if (current.number !== number || current.node_id !== queued.pullRequest.id) {
    throw new Error('Queued PR identity changed during dequeue');
  }
  if (current.state === 'open' && current.base.ref === RELEASE_BRANCH &&
      current.head.sha === queued.pullRequest.headRefOid && admitted(current)) {
    // Eligibility recovered after retirement. Keep the old cumulative SHAs
    // blocked; dequeue and re-enqueue manually to obtain fresh candidates.
    if (invalidationError) throw invalidationError;
    return [];
  }
  const before = await pullRequestQueueEntry(github, context, queued.pullRequest.id);
  if (before === null) {
    if (invalidationError) throw invalidationError;
    return [];
  }
  if (before?.id !== queued.id) {
    throw new Error('Queued PR entry changed during dequeue');
  }
  await github.graphql(`mutation DequeueReleasePR($id: ID!) {
    dequeuePullRequest(input: {id: $id}) { mergeQueueEntry { id } }
  }`, {id: queued.pullRequest.id});
  if (await pullRequestQueueEntry(github, context, queued.pullRequest.id) !== null) {
    throw new Error('Queued PR is still present after dequeue');
  }
  if (invalidationError) throw invalidationError;
  // Old cumulative SHAs stay pending. Fresh queue builds need fresh CI.
  return [];
}

async function queueHeads({github, context, recovery}) {
  if (context.eventName === 'schedule') {
    return /^[0-9a-f]{40}$/.test(recovery?.sha || '') ? [recovery.sha] : [];
  }
  if (context.eventName === 'workflow_run') {
    const run = context.payload.workflow_run;
    if (run?.event !== 'merge_group' ||
        !run.head_branch?.startsWith(QUEUE_BRANCH_PREFIX) ||
        run.head_repository?.full_name?.toLowerCase() !==
          `${context.repo.owner}/${context.repo.repo}`.toLowerCase()) return [];
    return /^[0-9a-f]{40}$/.test(run.head_sha || '') ? [run.head_sha] : [];
  }
  if (context.eventName === 'pull_request_target') {
    const eventPR = context.payload.pull_request;
    // The read-only discovery job already identified this affected SHA.
    // Preserve it even if a queue entry disappears before validation.
    if (recovery?.sha) {
      if (!/^[0-9a-f]{40}$/.test(recovery.sha)) {
        throw new Error('Invalid discovered release queue head');
      }
      return [recovery.sha];
    }
    if (eventPR?.base?.ref !== RELEASE_BRANCH) return [];
    const {entries} = await queueEntries(github, context);
    const ordered = orderedQueueEntries(entries);
    const matches = ordered.filter(entry => entry?.pullRequest?.number === eventPR.number);
    if (matches.length > 1) throw new Error('PR has ambiguous release merge queue heads');
    if (matches.length === 0) return [];
    const affected = ordered.filter(entry => entry.position >= matches[0].position);
    if (affected.some(entry => !/^[0-9a-f]{40}$/.test(entry?.headCommit?.oid || ''))) {
      throw new Error('Cannot identify all affected release merge queue heads');
    }
    return affected.map(entry => entry.headCommit.oid);
  }
  return [];
}

function queueTriggerApplies(trigger) {
  if (!trigger || typeof trigger !== 'object' || Array.isArray(trigger)) return false;
  const {types, ...filters} = trigger;
  if (types !== undefined && (!Array.isArray(types) ||
      types.length !== 1 || types[0] !== 'checks_requested')) {
    throw new Error('Unsupported release merge_group event types');
  }
  if (filters.paths || filters['paths-ignore']) {
    throw new Error('merge_group cannot use path filters');
  }
  return applicable(filters, RELEASE_BRANCH, []);
}

async function queueEvidence({github, context, sha, root, verifyCurrentRuns = false}) {
  const queue = await queueEntries(github, context);
  if (queue.configuration?.maximumEntriesToBuild !== 1 ||
      queue.configuration?.maximumEntriesToMerge !== 1 ||
      queue.configuration?.mergingStrategy !== 'ALLGREEN') {
    throw new Error('Release merge queue must build and merge one PR at a time with ALLGREEN checks');
  }
  const ordered = orderedQueueEntries(queue.entries);
  // The setting limits concurrently dispatched builds, not necessarily the
  // number of completed heads still visible. If more than one head exists,
  // fail before the expensive direct-run checks and require a live canary.
  if (ordered.filter(entry => entry?.headCommit?.oid != null).length > 1) {
    return {state: 'failure', reason: 'Multiple built queue heads exceed the CI publisher budget.'};
  }
  const matching = ordered.filter(entry => entry?.headCommit?.oid === sha);
  if (matching.length === 0) {
    // Actions completion can precede GraphQL queue entry visibility. A
    // missing required status blocks merge; the scheduled sweep will retry.
    return {state: 'pending', reason: 'Waiting for the merge queue entry to become visible.'};
  }
  if (matching.length !== 1 || !Number.isSafeInteger(matching[0]?.pullRequest?.number)) {
    return {state: 'failure', reason: 'Queue head ambiguously identifies a PR.'};
  }
  // A later temporary commit contains every PR ahead of it in the queue.
  // Recheck all those PRs, not just the PR named by this queue entry.
  const included = ordered.filter(entry => entry.position <= matching[0].position);
  const numbers = new Set();
  for (const entry of included) {
    const queued = entry.pullRequest;
    if (!Number.isSafeInteger(queued?.number) || queued.number <= 0 ||
        !/^[0-9a-f]{40}$/.test(queued.headRefOid || '') || numbers.has(queued.number)) {
      return {state: 'failure', reason: 'Queue prefix does not identify unique current PRs.'};
    }
    numbers.add(queued.number);
    const pr = await readPR(github, context, queued.number);
    if (pr.state !== 'open' || pr.base.ref !== RELEASE_BRANCH ||
        pr.head.sha !== queued.headRefOid || !admitted(pr)) {
      return {state: 'failure', reason: `Queued PR #${queued.number} is no longer admitted at this head.`};
    }
    const guard = await releaseWorkflowGuard(github, context, pr);
    if (!guard.passed) return {state: 'failure', reason: `Queued PR #${queued.number}: ${guard.reason}`};
  }
  const branch = (await github.rest.repos.getBranch({
    ...context.repo, branch: RELEASE_BRANCH,
  })).data;
  const baseSha = branch?.commit?.sha;
  if (!/^[0-9a-f]{40}$/.test(baseSha || '')) {
    throw new Error('Cannot identify the trusted release branch workflow policy');
  }
  const {inventory} = await loadBaseInventory({github, ...context.repo,
    pullRequest: {base: {ref: RELEASE_BRANCH, sha: baseSha,
      repo: {full_name: `${context.repo.owner}/${context.repo.repo}`}}}, root});
  const prWorkflows = inventory.workflows.filter(workflow =>
    workflow.pull_request !== null && workflow.disabled_for_migration !== true);
  if (prWorkflows.length === 0) {
    throw new Error('No release CI workflows found in trusted branch policy');
  }
  const byPath = new Map(inventory.workflows.map(workflow => [workflow.path, workflow]));
  for (const workflow of prWorkflows) {
    const queueWorkflow = byPath.get(RELEASE_QUEUE_EQUIVALENTS.get(workflow.path) || workflow.path);
    if (!queueWorkflow || !queueTriggerApplies(queueWorkflow.merge_group)) {
      return {state: 'failure', reason: `${workflow.path} has no release merge-group CI equivalent.`};
    }
  }
  // Every group-only workflow also counts. A newly added queue lane cannot
  // fail unnoticed just because it has no pull_request trigger.
  const expected = inventory.workflows.filter(workflow =>
    workflow.disabled_for_migration !== true && queueTriggerApplies(workflow.merge_group));
  const runs = await github.paginate(github.rest.actions.listWorkflowRunsForRepo, {
    ...context.repo, event: 'merge_group', head_sha: sha, per_page: 100,
  });
  if (runs.length >= 1000) throw new Error('Merge-group workflow history is truncated');
  const incomplete = [];
  const selected = [];
  for (const workflow of expected) {
    const matchingRuns = runs.filter(run => run.path === workflow.path &&
      run.event === 'merge_group' && run.head_sha === sha &&
      run.head_branch?.startsWith(QUEUE_BRANCH_PREFIX) &&
      run.head_repository?.full_name?.toLowerCase() ===
        `${context.repo.owner}/${context.repo.repo}`.toLowerCase());
    const attemptTime = run => Math.max(Date.parse(run.created_at) || 0,
      Date.parse(run.run_started_at) || 0);
    matchingRuns.sort((left, right) => attemptTime(right) - attemptTime(left) || right.id - left.id);
    const run = matchingRuns[0];
    if (!run || run.status !== 'completed') {
      incomplete.push(workflow.path);
    } else if (run.conclusion !== 'success') {
      return {state: 'failure', reason: `${workflow.path} completed with ${run.conclusion}.`};
    } else {
      selected.push({workflow, run});
    }
  }
  if (incomplete.length) return {state: 'pending', reason: `${incomplete.length} release CI workflows still running.`};
  if (!verifyCurrentRuns) {
    return {state: 'success', reason: 'All release merge-group CI and PR admission checks passed.'};
  }
  // The run listing can retain a successful earlier attempt after a rerun
  // starts. Only spend direct API calls once every listed lane is green.
  for (const {workflow, run} of selected) {
    const current = (await github.rest.actions.getWorkflowRun({
      ...context.repo, run_id: run.id,
    })).data;
    if (current.id !== run.id || current.head_sha !== sha ||
        current.path !== workflow.path || current.event !== 'merge_group' ||
        !current.head_branch?.startsWith(QUEUE_BRANCH_PREFIX) ||
        current.head_repository?.full_name?.toLowerCase() !==
          `${context.repo.owner}/${context.repo.repo}`.toLowerCase() ||
        !Number.isSafeInteger(current.run_attempt) || current.run_attempt < 1 ||
        !Number.isSafeInteger(run.run_attempt) ||
        current.run_attempt !== run.run_attempt || current.status !== 'completed') {
      incomplete.push(workflow.path);
    } else if (current.conclusion !== 'success') {
      return {state: 'failure', reason: `${workflow.path} completed with ${current.conclusion}.`};
    }
  }
  if (incomplete.length) return {state: 'pending', reason: `${incomplete.length} release CI workflows still running.`};
  return {state: 'success', reason: 'All release merge-group CI and PR admission checks passed.'};
}

async function queueStatus(github, context, sha, state, description) {
  // A status read can lag a previous write. Always append the desired state:
  // a stale pending read must never suppress revocation of newer success.
  await github.rest.repos.createCommitStatus({
    ...context.repo, sha, context: RELEASE_STATUS_CONTEXT, state,
    description: description.slice(0, 140),
    target_url: `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`,
  });
}

function queueInvalidationDescription(context) {
  const publisher = `${context.runId}.${context.runAttempt || process.env.GITHUB_RUN_ATTEMPT || 1}`;
  const run = context.payload.workflow_run;
  if (context.eventName === 'workflow_run' &&
      ['requested', 'in_progress'].includes(context.payload.action) &&
      run?.event === 'merge_group') {
    if (!Number.isSafeInteger(run.id) || !Number.isSafeInteger(run.run_attempt) ||
        run.run_attempt < 1) {
      // Still revoke the known SHA even when the webhook cannot identify its
      // attempt. Later evidence cannot prove this start has finished.
      return `queue-fence-invalid:${publisher}`;
    }
    return `queue-fence:${publisher}:${run.id}:${run.run_attempt}`;
  }
  return `queue-check:${publisher}`;
}

async function queueRerunFence(github, context, sha, requireCurrentMarker = true) {
  // Commit status history is append-only and returned newest first. Seeing
  // this publisher's pending marker at the front is a read-after-write
  // barrier for every earlier rerun marker on this SHA. If the status API
  // lags or a newer event has invalidated the SHA, leave it pending.
  const statuses = await github.paginate(github.rest.repos.listCommitStatusesForRef, {
    ...context.repo, ref: sha, per_page: 100,
  });
  const gateStatuses = statuses.filter(status => status.context === RELEASE_STATUS_CONTEXT);
  // Keep a write slot to revoke a success if the queue changes immediately
  // afterward. A stalled SHA must stop at pending before GitHub's per-context
  // status limit can strand a green result.
  if (gateStatuses.length >= 999) {
    return {state: 'pending', reason: 'Queue status history is full; recreate the queue entry.'};
  }
  if (requireCurrentMarker &&
      (gateStatuses[0]?.description !== queueInvalidationDescription(context) ||
       gateStatuses[0]?.state !== 'pending')) {
    return {state: 'pending', reason: 'Waiting for current queue invalidation to become visible.'};
  }
  if (!requireCurrentMarker && gateStatuses[0]?.state === 'pending') {
    return {state: 'pending', reason: 'A newer queue invalidation is pending.'};
  }
  const fenced = new Map();
  for (const status of gateStatuses) {
    if (/^queue-retired:\d+\.\d+$/.test(status.description || '')) {
      return {state: 'pending', reason: 'This cumulative queue SHA was retired for dequeue.'};
    }
    if (status.description?.startsWith('queue-retired')) {
      return {state: 'failure', reason: 'A retired queue marker is malformed.'};
    }
    if (/^queue-fence-invalid:\d+\.\d+$/.test(status.description || '')) {
      return {state: 'failure', reason: 'A merge-group start had no verifiable attempt.'};
    }
    const match = /^queue-fence:\d+\.\d+:(\d+):(\d+)$/.exec(status.description || '');
    if (status.description?.startsWith('queue-fence') && !match) {
      return {state: 'failure', reason: 'A merge-group fence marker is malformed.'};
    }
    if (match) {
      const runId = Number(match[1]);
      const attempt = Number(match[2]);
      if (!Number.isSafeInteger(runId) || !Number.isSafeInteger(attempt) || attempt < 1) {
        return {state: 'failure', reason: 'A merge-group fence marker is invalid.'};
      }
      fenced.set(`${runId}:${attempt}`, {runId, attempt});
    }
  }
  for (const {runId, attempt} of fenced.values()) {
    const request = {...context.repo, run_id: runId};
    const exact = (await github.rest.actions.getWorkflowRunAttempt({
      ...request, attempt_number: attempt,
    })).data;
    const current = (await github.rest.actions.getWorkflowRun(request)).data;
    if (exact.id !== runId || exact.run_attempt !== attempt ||
        exact.head_sha !== sha || exact.event !== 'merge_group' ||
        !exact.head_branch?.startsWith(QUEUE_BRANCH_PREFIX) ||
        exact.head_repository?.full_name?.toLowerCase() !==
          `${context.repo.owner}/${context.repo.repo}`.toLowerCase() ||
        current.id !== runId || !Number.isSafeInteger(current.run_attempt) ||
        current.run_attempt < attempt ||
        current.head_sha !== sha || current.event !== 'merge_group' ||
        !current.head_branch?.startsWith(QUEUE_BRANCH_PREFIX) ||
        current.head_repository?.full_name?.toLowerCase() !==
          `${context.repo.owner}/${context.repo.repo}`.toLowerCase() ||
        current.status !== 'completed' ||
        exact.status !== 'completed') {
      return {state: 'pending', reason: 'A fenced release workflow attempt is still running.'};
    }
    if (current.conclusion !== 'success') {
      return {state: 'failure', reason: 'The latest fenced release workflow attempt failed.'};
    }
    if (current.run_attempt === attempt && exact.conclusion !== 'success') {
      return {state: 'pending', reason: 'Exact and current attempt conclusions disagree.'};
    }
  }
  return {state: 'success'};
}

async function reconcileQueue({github, context, core, recovery, alreadyPending = false,
  root = process.env.GITHUB_WORKSPACE}) {
  const shas = await queueHeads({github, context, recovery});
  if (!alreadyPending) {
    // Direct callers revoke every affected build before slower validation.
    for (const sha of shas) {
      await queueStatus(github, context, sha, 'pending', queueInvalidationDescription(context));
    }
  }
  for (const sha of shas) {
    // A requested/in_progress workflow_run invalidates a previous success.
    // Actions' run-list API may still show the prior completed attempt here.
    if (context.eventName === 'workflow_run' &&
        ['requested', 'in_progress'].includes(context.payload.action)) continue;
    try {
      let result = await queueEvidence({github, context, sha, root});
      core.info(JSON.stringify(result));
      if (result.state === 'success') {
        // A label, PR head, queue entry, or CI rerun may change during reads.
        result = await queueEvidence({github, context, sha, root, verifyCurrentRuns: true});
      }
      if (result.state === 'success') {
        result = await queueRerunFence(github, context, sha);
        if (result.state === 'success') {
          result = {state: 'success', reason: 'All release merge-group CI and PR admission checks passed.'};
        }
      }
      await queueStatus(github, context, sha, result.state, result.reason);
      if (result.state === 'success') {
        let after = await queueEvidence({github, context, sha, root});
        if (after.state === 'success') {
          after = await queueRerunFence(github, context, sha, false);
        }
        if (after.state !== 'success') await queueStatus(github, context, sha,
          'failure', 'Release queue or CI changed during publication.');
      }
    } catch (error) {
      await queueStatus(github, context, sha, 'failure',
        'Cannot verify release queue CI; inspect CI Check and retry.');
      throw error;
    }
  }
}

module.exports = {recoveryCandidates, eligible, admitted, snapshot, resolve, freshAfter,
  prepare, finalize, queueRecoveryCandidates, queueEventCandidates, queueHeads,
  queueTriggerApplies,
  queueEvidence, queueStatus, queueInvalidationDescription, queueRerunFence, reconcileQueue};
