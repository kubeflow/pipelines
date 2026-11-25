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

const {createHash} = require('node:crypto');

// Bind the publisher's exact API snapshot, including attempt transitions. This
// must be called on evidence already read for validation, never a later lookup.
function workflowAttempts(runs, head) {
  if (!Array.isArray(runs) || !runs.length || runs.length >= 1000) {
    throw new Error('Incomplete admission workflow history');
  }
  const ids = new Set();
  for (const run of runs) {
    const active = ['queued', 'requested', 'waiting', 'in_progress', 'pending'].includes(run.status);
    const completed = run.status === 'completed' &&
      ['success', 'failure', 'neutral', 'cancelled', 'skipped', 'timed_out', 'action_required', 'stale', 'startup_failure'].includes(run.conclusion);
    if (!Number.isSafeInteger(run.id) || run.id <= 0 || ids.has(run.id) ||
        !Number.isSafeInteger(run.run_attempt) || run.run_attempt <= 0 ||
        run.event !== 'pull_request' || run.head_sha !== head ||
        !(completed || (active && run.conclusion === null))) {
      throw new Error('Invalid admission workflow attempt');
    }
    ids.add(run.id);
  }
  return createHash('sha256').update(JSON.stringify(runs.map(run =>
    [run.id, run.run_attempt, run.status, run.conclusion]).sort((a, b) => a[0] - b[0]))).digest('hex');
}

function publishedAttempts(status, repository) {
  const match = new RegExp(`^https://github\\.com/${repository.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}/actions/runs/[1-9][0-9]*#queue-attempts=([0-9a-f]{64})$`).exec(status?.target_url || '');
  return match?.[1] || null;
}

module.exports = {workflowAttempts, publishedAttempts};
