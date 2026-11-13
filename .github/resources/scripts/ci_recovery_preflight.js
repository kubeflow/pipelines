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

const {applicable, loadBaseInventory, baseArrivedAt} = require('./ci_expected_workflows');
const {currentWorkflowRuns} = require('./ci_check_runs');
const SHA = /^[0-9a-f]{40}$/;

function recoveryPreflight({github, context, root}) {
  // Discovery can only retain an existing non-green result, never publish or
  // authorize success. Share base reads within this sweep; every later sweep
  // and the full publisher independently reads current refs, PRs and attempts.
  const bases = new Map();
  async function baseEvidence(pr, queueAdmission) {
    const development = ['master', 'main'].includes(pr.base.ref);
    const key = JSON.stringify([pr.base.ref, development ? null : pr.base.sha, queueAdmission]);
    if (!bases.has(key)) bases.set(key, (async () => {
      let sha = pr.base.sha;
      if (development) {
        const {data: ref} = await github.rest.git.getRef({
          ...context.repo, ref: `heads/${pr.base.ref}`,
        });
        if (ref?.ref !== `refs/heads/${pr.base.ref}` || ref.object?.type !== 'commit') {
          throw Error('Invalid recovery base ref');
        }
        sha = ref.object.sha;
      }
      if (!SHA.test(sha || '')) throw Error('Invalid recovery base SHA');
      const {inventory} = await loadBaseInventory({github, ...context.repo,
        pullRequest: pr, baseSha: sha, root, cacheRunId: context.runId});
      const required = inventory.workflows.filter(workflow => {
        const trigger = workflow.pull_request;
        if (!trigger || Object.hasOwn(trigger, 'paths') || Object.hasOwn(trigger, 'paths-ignore') ||
            workflow.disabled_for_migration === true ||
            (development && workflow.path === '.github/workflows/legacy-v2-api-integration-tests.yml')) return false;
        return applicable(trigger, pr.base.ref, []);
      }).map(workflow => workflow.path);
      // Arrival is needed only for development freshness, never queue admission
      // or a frozen release base. Missing provenance falls back to full recovery.
      let arrivedAt = null;
      if (development && !queueAdmission) {
        const arrival = await baseArrivedAt({github, ...context.repo, baseSha: sha, baseRef: pr.base.ref});
        if (arrival.status !== 'arrived') throw Error('Cannot establish recovery base arrival');
        arrivedAt = arrival.arrivedAt;
      }
      return {required, arrivedAt};
    })());
    return bases.get(key);
  }
  return async (pr, status, queueAdmission) => {
    if (!['failure', 'pending'].includes(status?.state)) return false;
    try {
      if (!SHA.test(pr.head?.sha || '') || !pr.head?.ref || !pr.head.repo?.full_name ||
          pr.base?.repo?.full_name !== `${context.repo.owner}/${context.repo.repo}`) return false;
      const {required, arrivedAt} = await baseEvidence(pr, queueAdmission);
      if (!required.length) return false;
      const runs = await currentWorkflowRuns({github, ...context.repo, sha: pr.head.sha, event: 'pull_request'});
      // An endpoint or identity mismatch is not evidence that an expected lane
      // is absent. Active attempts and all ambiguity receive full recovery.
      if (runs.some(run => run.event !== 'pull_request' || run.head_branch !== pr.head.ref ||
          run.head_repository?.full_name !== pr.head.repo.full_name ||
          run.repository?.full_name !== `${context.repo.owner}/${context.repo.repo}` ||
          run.status !== 'completed')) return false;
      let stale = false;
      for (const path of required) {
        const matching = runs.filter(run => run.path === path);
        if (matching.length > 1) return false;
        const run = matching[0];
        // Keep the pending-to-failure registration deadline and failed-run
        // transitions in the full publisher. Only existing failures can stay
        // failed cheaply on missing or unsuccessful required lanes.
        if (!run || run.conclusion !== 'success') {
          if (status.state === 'failure') return true;
          return false;
        }
        if (arrivedAt !== null && Date.parse(run.created_at) <= arrivedAt) stale = true;
      }
      return stale;
    } catch (_) {
      // This optimization cannot suppress recovery on missing or malformed data.
      return false;
    }
  };
}

module.exports = {recoveryPreflight};
