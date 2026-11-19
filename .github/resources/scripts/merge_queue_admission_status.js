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

const STATUS = 'merge-queue-admission';

// Diagnostic only: this context must never be a required check or CI evidence.
// Its failure must not prevent admission or revoke an already queued candidate.
async function publishAdmissionStatus({github, context, pr, state = 'pending', description, core}) {
  if (!/^[0-9a-f]{40}$/.test(pr?.head?.sha || '') || !Number.isSafeInteger(pr?.number)) return;
  try {
    const {data} = await github.rest.repos.getCombinedStatusForRef({
      ...context.repo, ref: pr.head.sha, per_page: 100,
    });
    if (!Array.isArray(data?.statuses)) throw new Error('Incomplete admission diagnostic statuses');
    const previous = data.statuses.find(status => status.context === STATUS);
    // Use a stable PR link so scheduled sweeps do not create duplicate statuses.
    const target_url = `https://github.com/${context.repo.owner}/${context.repo.repo}/pull/${pr.number}`;
    description = description.slice(0, 140);
    if (previous?.state === state && previous.description === description && previous.target_url === target_url) return;
    await github.rest.repos.createCommitStatus({...context.repo, sha: pr.head.sha,
      context: STATUS, state, description, target_url});
  } catch (error) {
    (core?.warning || core?.info || (() => {}))(`PR #${pr.number}: could not publish admission diagnostic: ${error.message}`);
  }
}

async function unresolvedConversations(github, context, pr) {
  // Mirror the effective rules instead of imposing a new review requirement.
  const rules = await github.paginate('GET /repos/{owner}/{repo}/rules/branches/{branch}', {
    ...context.repo, branch: pr.base.ref, per_page: 100,
  });
  if (!Array.isArray(rules)) throw new Error('Incomplete admission rules');
  const policy = await github.graphql(`query AdmissionReviewPolicy($owner:String!, $repo:String!, $branch:String!) {
    repository(owner:$owner, name:$repo) { nameWithOwner ref(qualifiedName:$branch) {
      name refUpdateRule { requiresConversationResolution }
    } }
  }`, {...context.repo, branch: `refs/heads/${pr.base.ref}`});
  const ref = policy?.repository?.ref;
  if (policy?.repository?.nameWithOwner !== `${context.repo.owner}/${context.repo.repo}` ||
      ref?.name !== pr.base.ref || (ref.refUpdateRule !== null &&
      typeof ref.refUpdateRule?.requiresConversationResolution !== 'boolean')) {
    throw new Error('Incomplete admission conversation policy');
  }
  const required = ref.refUpdateRule?.requiresConversationResolution ||
    rules.some(rule => rule.type === 'pull_request' && rule.parameters?.required_review_thread_resolution === true);
  if (!required) return 0;
  let cursor = null, total = null, count = 0, unresolved = 0;
  const cursors = new Set();
  for (let page = 0; page < 10; page++) {
    const result = await github.graphql(`query AdmissionConversations($owner:String!, $repo:String!, $number:Int!, $cursor:String) {
      repository(owner:$owner, name:$repo) { nameWithOwner pullRequest(number:$number) {
        id headRefOid reviewThreads(first:100, after:$cursor) {
          totalCount nodes { isResolved } pageInfo { hasNextPage endCursor }
        }
      } }
    }`, {...context.repo, number: pr.number, cursor});
    const current = result?.repository?.pullRequest;
    const threads = current?.reviewThreads;
    if (result?.repository?.nameWithOwner !== `${context.repo.owner}/${context.repo.repo}` ||
        current?.id !== pr.node_id || current.headRefOid !== pr.head.sha ||
        !Number.isSafeInteger(threads?.totalCount) || threads.totalCount < 0 || threads.totalCount >= 1000 ||
        (total !== null && total !== threads.totalCount) || !Array.isArray(threads.nodes) ||
        threads.nodes.length > 100 || threads.nodes.some(thread => typeof thread?.isResolved !== 'boolean') ||
        typeof threads.pageInfo?.hasNextPage !== 'boolean') {
      throw new Error('Incomplete or changed admission review conversations');
    }
    total = threads.totalCount;
    count += threads.nodes.length;
    unresolved += threads.nodes.filter(thread => !thread.isResolved).length;
    if (count > total) throw new Error('Incomplete admission review conversations');
    if (!threads.pageInfo.hasNextPage) {
      if (count !== total) throw new Error('Incomplete admission review conversations');
      return unresolved;
    }
    cursor = threads.pageInfo.endCursor;
    if (!threads.nodes.length || typeof cursor !== 'string' || !cursor || cursors.has(cursor)) {
      throw new Error('Invalid admission review pagination');
    }
    cursors.add(cursor);
  }
  throw new Error('Admission review pagination limit exceeded');
}

// Only recognized policy rejections are normal waiting states. Transport and
// unknown errors retain fail-closed cleanup and shared-job failure handling.
function enqueueBlocker(error) {
  const errors = error?.errors;
  if (!Array.isArray(errors) || !errors.length) return null;
  const messages = [];
  for (const item of errors) {
    if (item.type !== 'UNPROCESSABLE' || !Array.isArray(item.path) ||
        item.path[0] !== 'enqueuePullRequest' || typeof item.message !== 'string') return null;
    if (/all (?:comments|conversations) must be resolved/i.test(item.message)) {
      messages.push('Waiting for unresolved review conversations.');
    } else if (/required (?:status )?checks?.*(?:not|missing|pending|fail)/i.test(item.message)) {
      messages.push('Waiting for required GitHub checks; see PR checks.');
    } else if (/merge conflict|not mergeable/i.test(item.message)) {
      messages.push('GitHub reports the PR is not mergeable; see PR merge requirements.');
    } else return null;
  }
  return [...new Set(messages)].join(' ');
}

module.exports = {publishAdmissionStatus, unresolvedConversations, enqueueBlocker};
