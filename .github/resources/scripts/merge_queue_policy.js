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
      policy.enabled_branches.some(branch => !['master', 'main'].includes(branch)) ||
      new Set(policy.enabled_branches).size !== policy.enabled_branches.length) {
    throw new Error('Invalid merge queue policy; enable only master or main with policy version 1.');
  }
  return policy;
}

async function queueEnabled({github, context, branch, root}) {
  if (!loadPolicy(root).enabled_branches.includes(branch)) return false;
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

module.exports = {loadPolicy, queueEnabled};
