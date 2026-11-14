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

const fs = require('node:fs');
const path = require('node:path');
const DEFAULT_ROOT = path.resolve(__dirname, '../../..');

function loadPolicy(root = DEFAULT_ROOT) {
  // This file comes from the trusted publisher checkout, never a PR checkout.
  const policy = JSON.parse(fs.readFileSync(path.join(root, '.github/merge-queue-policy.json'), 'utf8'));
  if (!policy || Array.isArray(policy) || policy.version !== 1 ||
      Object.keys(policy).some(key => !['version', 'enabled_branches'].includes(key)) ||
      !Array.isArray(policy.enabled_branches) ||
      policy.enabled_branches.some(branch => !['master', 'main', '~DEFAULT_BRANCH'].includes(branch)) ||
      new Set(policy.enabled_branches).size !== policy.enabled_branches.length ||
      (policy.enabled_branches.includes('~DEFAULT_BRANCH') && policy.enabled_branches.length !== 1)) {
    throw new Error('Invalid merge queue policy; use master/main or the exclusive ~DEFAULT_BRANCH selector with policy version 1.');
  }
  return policy;
}

async function enabledBranches({github, context, root}) {
  const branches = loadPolicy(root).enabled_branches;
  if (!branches.includes('~DEFAULT_BRANCH')) return branches;
  // Read trusted repository metadata at every boundary, never PR payload data
  // or a cached default: a rename can occur between admission and publication.
  const {data} = await github.rest.repos.get({...context.repo});
  if (data?.full_name !== `${context.repo.owner}/${context.repo.repo}` ||
      !['master', 'main'].includes(data?.default_branch)) {
    throw new Error('Cannot resolve merge queue default branch to trusted master or main.');
  }
  return [data.default_branch];
}

async function queueEnabled({github, context, branch, root}) {
  if (!['master', 'main'].includes(branch)) return false;
  if (!(await enabledBranches({github, context, root})).includes(branch)) return false;
  // This endpoint returns rules from active rulesets applicable to this branch,
  // including organization rules. A configured but unenforced queue is insufficient.
  const rules = await github.paginate('GET /repos/{owner}/{repo}/rules/branches/{branch}', {
    ...context.repo, branch, per_page: 100,
  });
  if (!Array.isArray(rules) || !rules.some(rule => rule?.type === 'merge_queue')) {
    throw new Error(`Merge queue policy requires an enforced merge_queue rule for ${branch}.`);
  }
  return true;
}

module.exports = {loadPolicy, enabledBranches, queueEnabled};
