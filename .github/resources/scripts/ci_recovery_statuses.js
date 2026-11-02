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

const BATCH_SIZE = 50;
const STATES = new Set(['ERROR', 'EXPECTED', 'FAILURE', 'PENDING', 'SUCCESS']);

async function readRecoveryStatuses({github, owner, repo, heads}) {
  if (!Array.isArray(heads) || heads.some(head => typeof head !== 'string' ||
      !/^[0-9a-f]{40}$/.test(head))) throw new Error('Recovery requires exact commit SHAs');
  const uniqueHeads = [...new Set(heads)];
  const statuses = new Map();
  // Query the upstream repository, even for fork heads: this is where the
  // trusted publisher writes ci-passed. Bound query size without concurrent bursts.
  for (let offset = 0; offset < uniqueHeads.length; offset += BATCH_SIZE) {
    const batch = uniqueHeads.slice(offset, offset + BATCH_SIZE);
    const variables = {owner, repo};
    const declarations = batch.map((head, index) => {
      variables[`sha${index}`] = head;
      return `$sha${index}: GitObjectID!`;
    });
    const fields = batch.map((head, index) => `head${index}: object(oid: $sha${index}) {
      __typename
      ... on Commit { oid status { context(name: "ci-passed") { context state description } } }
    }`);
    const result = await github.graphql(`query RecoveryStatuses($owner: String!, $repo: String!,
      ${declarations.join(', ')}) {
      repository(owner: $owner, name: $repo) { nameWithOwner ${fields.join('\n')} }
    }`, variables);
    // Octokit throws for GraphQL errors, including partial data; also reject
    // explicit error envelopes from alternative clients rather than using them.
    if (result?.errors !== undefined) throw new Error('Incomplete recovery status response');
    const repository = result?.repository;
    if (typeof repository?.nameWithOwner !== 'string' ||
        repository.nameWithOwner.toLowerCase() !== `${owner}/${repo}`.toLowerCase()) {
      throw new Error('Recovery status repository mismatch');
    }
    for (const [index, head] of batch.entries()) {
      const commit = repository[`head${index}`];
      if (commit?.__typename !== 'Commit' || commit.oid !== head ||
          !Object.hasOwn(commit, 'status')) throw new Error('Incomplete recovery commit response');
      if (commit.status === null) {
        statuses.set(head, null);
        continue;
      }
      const status = commit.status?.context;
      if (status === null) {
        statuses.set(head, null);
        continue;
      }
      if (status?.context !== 'ci-passed' || !STATES.has(status.state) ||
          !(status.description === null || typeof status.description === 'string')) {
        throw new Error('Invalid recovery status context');
      }
      statuses.set(head, {state: status.state.toLowerCase(), description: status.description});
    }
  }
  return statuses;
}

module.exports = {readRecoveryStatuses};
