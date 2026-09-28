// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

'use strict';

const EXCLUDED = /^(?:check_ci_status(?: \(.*\))?|Cleanup artifacts|Upload results|Agent|Prepare)$/;
const ACTIVE = new Set(['queued', 'in_progress', 'waiting', 'requested', 'pending']);
const PASSED = new Set(['success', 'skipped', 'neutral']);
const FAILED = new Set(['failure', 'cancelled', 'timed_out', 'action_required', 'stale', 'startup_failure']);
const positiveId = value => Number.isSafeInteger(value) && value > 0;
const timestamp = value => typeof value === 'string' && Number.isFinite(Date.parse(value));

function validateState(item, label) {
  if (item.status === 'completed') {
    if (!PASSED.has(item.conclusion) && !FAILED.has(item.conclusion)) {
      throw new Error(`${label}: unknown completed conclusion`);
    }
  } else if (!ACTIVE.has(item.status) || item.conclusion !== null) {
    throw new Error(`${label}: invalid status or conclusion`);
  }
}

// Validate total_count on every page instead of approving a partial snapshot.
// The head-filtered Actions endpoint caps results at 1,000. Reject the boundary
// for every collection so an API limit never silently omits evidence.
async function readPages(route, field, options) {
  const records = [];
  const ids = new Set();
  let total = null;
  for (let page = 1; page <= 10; page++) {
    const response = await route({...options, per_page: 100, page});
    const data = response?.data;
    if (!Number.isSafeInteger(data?.total_count) || data.total_count < 0 ||
        data.total_count >= 1000 || !Array.isArray(data[field]) || data[field].length > 100 ||
        (total !== null && total !== data.total_count)) {
      throw new Error(`${field}: incomplete or changing API collection`);
    }
    total = data.total_count;
    for (const record of data[field]) {
      if (!positiveId(record?.id) || ids.has(record.id)) {
        throw new Error(`${field}: missing or duplicate record ID`);
      }
      ids.add(record.id);
      records.push(record);
    }
    if (records.length === total) return records;
    if (records.length > total || data[field].length < 100) {
      throw new Error(`${field}: incomplete API page`);
    }
  }
  throw new Error(`${field}: pagination limit reached`);
}

function validateRun(run, sha) {
  if (!positiveId(run.workflow_id) || !positiveId(run.check_suite_id) ||
      !positiveId(run.run_attempt) || run.head_sha !== sha ||
      typeof run.event !== 'string' || !run.event || !timestamp(run.created_at) ||
      (run.run_started_at !== null && !timestamp(run.run_started_at)) ||
      (run.run_attempt > 1 && !timestamp(run.run_started_at))) {
    throw new Error(`Workflow run ${run.id}: incomplete identity or attempt metadata`);
  }
  validateState(run, `Workflow run ${run.id}`);
}

function attemptTime(run) {
  return Math.max(Date.parse(run.created_at), Date.parse(run.run_started_at) || 0);
}

function laterRun(left, right) {
  return attemptTime(left) > attemptTime(right) ||
    (attemptTime(left) === attemptTime(right) &&
      (left.id > right.id || (left.id === right.id && left.run_attempt > right.run_attempt)));
}

function checkIdForJob(job, run, owner, repo) {
  if (job.run_id !== run.id || job.run_attempt !== run.run_attempt ||
      typeof job.name !== 'string' || !job.name) {
    throw new Error(`Workflow run ${run.id}: incomplete current-attempt job metadata`);
  }
  validateState(job, `Job ${job.id}`);
  // A queued job may be visible before its check is published. Missing check
  // identity cannot authorize success, but is still an in-flight state.
  if (job.check_run_url === null && ACTIVE.has(job.status)) return null;
  if (typeof job.check_run_url !== 'string') throw new Error(`Job ${job.id}: missing check URL`);
  let url;
  try {
    url = new URL(job.check_run_url);
  } catch {
    throw new Error(`Job ${job.id}: invalid check URL`);
  }
  const prefix = `/repos/${owner}/${repo}/check-runs/`;
  const value = url.pathname.startsWith(prefix) ? url.pathname.slice(prefix.length) : '';
  if (url.protocol !== 'https:' || !/^[1-9][0-9]*$/.test(value) || !positiveId(Number(value))) {
    throw new Error(`Job ${job.id}: check URL does not identify this repository`);
  }
  return Number(value);
}

async function verifyCheckRuns({github, owner, repo, sha}) {
  const failures = [];
  const pending = [];
  const addState = (item, label) => {
    if (item.status !== 'completed') pending.push(`${label}: ${item.status}`);
    else if (FAILED.has(item.conclusion)) failures.push(`${label}: ${item.conclusion}`);
  };
  let checks = [];
  let runs = [];
  const results = await Promise.allSettled([
    readPages(options => github.rest.checks.listForRef(options), 'check_runs', {
      owner, repo, ref: sha, filter: 'all',
    }),
    readPages(options => github.rest.actions.listWorkflowRunsForRepo(options), 'workflow_runs', {
      owner, repo, head_sha: sha,
    }),
  ]);
  for (const [index, result] of results.entries()) {
    if (result.status === 'rejected') {
      failures.push(`Cannot verify ${index === 0 ? 'check runs' : 'workflow runs'}: ${result.reason?.message || 'API error'}`);
    } else if (index === 0) checks = result.value;
    else runs = result.value;
  }

  const bySuite = new Map();
  const latestRuns = new Map();
  for (const run of runs) {
    try {
      validateRun(run, sha);
      if (bySuite.has(run.check_suite_id)) throw new Error(`Workflow run ${run.id}: duplicate suite identity`);
      bySuite.set(run.check_suite_id, run);
      const previous = latestRuns.get(run.workflow_id);
      if (!previous || laterRun(run, previous)) latestRuns.set(run.workflow_id, run);
    } catch (error) {
      failures.push(error.message);
    }
  }

  const relevantWorkflowIds = new Set();
  const candidates = [];
  for (const check of checks) {
    try {
      if (typeof check.name !== 'string' || !check.name) throw new Error(`Check ${check.id}: missing name`);
      if (EXCLUDED.test(check.name)) continue;
      if (!positiveId(check.app?.id) || typeof check.app.slug !== 'string' || !check.app.slug ||
          !positiveId(check.check_suite?.id) || check.head_sha !== sha) {
        throw new Error(`Check ${check.id}: incomplete identity metadata`);
      }
      validateState(check, `Check ${check.name}`);
      if (check.app.slug === 'github-actions') {
        const run = bySuite.get(check.check_suite.id);
        if (!run) throw new Error(`Check ${check.name}: workflow run metadata is missing`);
        relevantWorkflowIds.add(run.workflow_id);
        if (latestRuns.get(run.workflow_id)?.id !== run.id) continue;
      }
      candidates.push(check);
    } catch (error) {
      failures.push(error.message);
    }
  }

  // Reruns reuse a suite ID. Fetch actual jobs only for rerun attempts, rather
  // than inferring an attempt from check IDs or timestamp coincidence. Earlier
  // successful jobs remain valid during failed-jobs-only reruns.
  const attemptChecks = new Map();
  for (const workflowId of relevantWorkflowIds) {
    const run = latestRuns.get(workflowId);
    addState(run, `Workflow run ${run.id}, attempt ${run.run_attempt}`);
    if (run.run_attempt === 1) continue;
    try {
      const jobs = await readPages(options => github.rest.actions.listJobsForWorkflowRunAttempt(options), 'jobs', {
        owner, repo, run_id: run.id, attempt_number: run.run_attempt,
      });
      const currentIds = new Set();
      for (const job of jobs) {
        const id = checkIdForJob(job, run, owner, repo);
        if (id === null) {
          if (!EXCLUDED.test(job.name)) pending.push(`Job ${job.name}: current-attempt check has not registered`);
          continue;
        }
        if (currentIds.has(id)) throw new Error(`Workflow run ${run.id}: duplicate job check identity`);
        currentIds.add(id);
        if (EXCLUDED.test(job.name)) continue;
        addState(job, `Job ${job.name}, attempt ${run.run_attempt}`);
        const check = candidates.find(item => item.id === id);
        if (!check || check.check_suite.id !== run.check_suite_id ||
            check.name !== job.name || check.app.slug !== 'github-actions' ||
            check.status !== job.status || check.conclusion !== job.conclusion) {
          pending.push(`Job ${job.name}: current-attempt check evidence is not synchronized`);
        }
      }
      if (jobs.length === 0) pending.push(`Workflow run ${run.id}: current-attempt jobs have not registered`);
      attemptChecks.set(run.id, currentIds);
    } catch (error) {
      failures.push(`Cannot verify workflow run ${run.id} attempt: ${error.message}`);
    }
  }

  const latestChecks = new Map();
  for (const check of candidates) {
    if (check.app.slug === 'github-actions') {
      const run = bySuite.get(check.check_suite.id);
      const currentIds = attemptChecks.get(run.id);
      if (currentIds && !currentIds.has(check.id) &&
          !(check.status === 'completed' && PASSED.has(check.conclusion))) continue;
    }
    const key = JSON.stringify([check.name, check.app.id]);
    if (!latestChecks.has(key) || latestChecks.get(key).id < check.id) latestChecks.set(key, check);
  }
  for (const check of latestChecks.values()) addState(check, `Check ${check.name} (app ${check.app.id})`);
  if (latestChecks.size === 0) pending.push('No eligible check runs have registered');
  const reasons = [...new Set(failures.length ? failures.concat(pending) : pending)];
  return {state: failures.length ? 'failure' : pending.length ? 'pending' : 'success', reasons};
}

module.exports = {verifyCheckRuns};
