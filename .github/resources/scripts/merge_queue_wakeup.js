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

const {policyReason} = require('./merge_queue_admission');

// Called only with finalize's post-publication verified success. GITHUB_TOKEN
// status writes do not trigger workflows; workflow_dispatch explicitly wakes the
// serialized queue writer without waiting for unrelated recovery matrix jobs.
async function wakeAdmission({github, context, core, pr}) {
  if (!pr || policyReason(pr, `${context.repo.owner}/${context.repo.repo}`, pr.base?.ref)) return;
  try {
    const {data} = await github.rest.repos.get(context.repo);
    if (!['main', 'master'].includes(data.default_branch)) throw new Error('Invalid default branch');
    await github.rest.actions.createWorkflowDispatch({...context.repo,
      workflow_id: 'merge-queue-recovery.yml', ref: data.default_branch,
      inputs: {pr_number: String(pr.number), head_sha: pr.head.sha}});
  } catch (error) {
    // Notification failure is not failed CI. The next surviving writer discovers
    // ready PRs, so a lost notification never requires another status transition.
    core.warning(`Queue admission wakeup failed; scheduled recovery will retry: ${error.message}`);
  }
}

module.exports = {wakeAdmission};
