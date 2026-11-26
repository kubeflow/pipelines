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

const {invalidRunMetadata} = require('./ci_expected_workflows');
const WORKFLOW = '.github/workflows/pr-gate.yml';
const RECEIPT = 'ci-pr-gate-retry/';
const COOLDOWN_MS = 15 * 60 * 1000;
const MAX_RETRIES_PER_HEAD = 2;

function membershipFailure(jobs, run) {
  if (!Array.isArray(jobs) || jobs.length !== 1) return false;
  const job = jobs[0];
  if (job.run_id !== run.id || job.head_sha !== run.head_sha ||
      job.name !== 'check-pr-author' || job.status !== 'completed' ||
      job.conclusion !== 'failure' || !Array.isArray(job.steps)) return false;
  const failed = job.steps.filter(step => step.conclusion === 'failure');
  if (failed.length !== 1 || failed[0].name !== 'Look up PR author in Kubeflow internal-acls') return false;
  const admission = job.steps.filter(step =>
    ['Check linked issue admission for external contributors',
      'Approve pending workflow runs after admission'].includes(step.name));
  return admission.length === 2 && admission.every(step => step.conclusion === 'skipped') &&
    job.steps.every(step => ['success', 'skipped', 'failure'].includes(step.conclusion));
}

// The caller supplies fresh head/approval/hold validation at both write boundaries.
// Membership lookup failures are infrastructure failures, not evidence of a
// quota error specifically. Never rerun failed issue admission or test work.
async function recoverPRGate({github, context, pr, revalidate, core, now = () => Date.now()}) {
  const repository = `${context.repo.owner}/${context.repo.repo}`;
  if (!Number.isSafeInteger(pr.number) || !/^[0-9a-f]{40}$/.test(pr.head?.sha || '') ||
      typeof pr.head.ref !== 'string' || !pr.head.ref ||
      typeof pr.head.repo?.full_name !== 'string' ||
      !pr.head.repo.full_name.startsWith(`${pr.head.repo.owner?.login}/`) ||
      typeof revalidate !== 'function') return false;
  const matches = run => run?.path === WORKFLOW && run.event === 'pull_request_target' &&
    run.head_sha === pr.head.sha && run.head_branch === pr.head.ref &&
    run.repository?.full_name === repository &&
    run.head_repository?.full_name === pr.head.repo.full_name;
  const retryable = run => matches(run) && !invalidRunMetadata(run) &&
    run.run_attempt === 1 && run.status === 'completed' && run.conclusion === 'failure' &&
    Number.isFinite(Date.parse(run.updated_at)) && Date.parse(run.updated_at) >= Date.parse(run.created_at) &&
    Date.parse(run.updated_at) <= now() - COOLDOWN_MS;
  // Match ci_check_runs.js: reruns can supersede runs created more recently.
  const attemptTime = run => Math.max(Date.parse(run.created_at), Date.parse(run.run_started_at) || 0);
  const authoritativeRun = async () => {
    const runs = await github.paginate(github.rest.actions.listWorkflowRunsForRepo, {
      ...context.repo, event: 'pull_request_target', head_sha: pr.head.sha, per_page: 100,
    });
    if (!Array.isArray(runs) || runs.length >= 1000) throw new Error('Incomplete PR Gate workflow discovery');
    const candidates = runs.filter(matches);
    if (candidates.some(run => invalidRunMetadata(run))) throw new Error('Invalid PR Gate workflow metadata');
    candidates.sort((a, b) => attemptTime(b) - attemptTime(a) || b.id - a.id || b.run_attempt - a.run_attempt);
    return candidates[0];
  };
  const run = await authoritativeRun();
  // A newer success or running attempt supersedes the historical incident.
  if (!run || !retryable(run)) return false;
  const statuses = await github.paginate(github.rest.repos.listCommitStatusesForRef, {
    ...context.repo, ref: pr.head.sha, per_page: 100,
  });
  if (!Array.isArray(statuses) || statuses.length >= 1000) throw new Error('Incomplete PR Gate retry history');
  const consumed = new Set(statuses.filter(status => status.context?.startsWith(RECEIPT))
    .map(status => status.context));
  if (consumed.size >= MAX_RETRIES_PER_HEAD) return false;
  const prs = await github.paginate(github.rest.pulls.list, {
    ...context.repo, state: 'open', head: `${pr.head.repo.owner.login}:${pr.head.ref}`, per_page: 100,
  });
  if (!Array.isArray(prs) || prs.length >= 1000) throw new Error('Incomplete PR Gate head discovery');
  const heads = prs.filter(candidate => candidate.head?.sha === pr.head.sha &&
    candidate.head.ref === pr.head.ref && candidate.head.repo?.full_name === pr.head.repo.full_name);
  if (heads.length !== 1 || heads[0].number !== pr.number) return false;
  // Retrying a historical failure would make it authoritative again. Only the
  // current run may consume a retry, even if its receipt already exists.
  const statusContext = `${RECEIPT}${run.id}`;
  if (consumed.has(statusContext)) return false;
  const jobs = await github.paginate(github.rest.actions.listJobsForWorkflowRunAttempt, {
    ...context.repo, run_id: run.id, attempt_number: run.run_attempt, per_page: 100,
  });
  if (!membershipFailure(jobs, run)) return false;
  const {data: current} = await github.rest.actions.getWorkflowRun({...context.repo, run_id: run.id});
  if (current.id !== run.id || !retryable(current) || !await revalidate()) return false;
  const authoritative = await authoritativeRun();
  if (authoritative?.id !== run.id || !retryable(authoritative)) return false;
  // Serialize with queue admission and reserve before requesting a rerun.
  // A lost response consumes the budget rather than causing an endless retry.
  await github.rest.repos.createCommitStatus({...context.repo, sha: pr.head.sha,
    context: statusContext, state: 'pending', description: `PR Gate run ${run.id}: automatic infrastructure retry reserved.`});
  if (!await revalidate()) return false;
  const {data: fresh} = await github.rest.actions.getWorkflowRun({...context.repo, run_id: run.id});
  if (fresh.id !== run.id || !retryable(fresh)) return false;
  const latest = await authoritativeRun();
  if (latest?.id !== run.id || !retryable(latest)) return false;
  await github.rest.actions.reRunWorkflowFailedJobs({...context.repo, run_id: run.id});
  core.info(`Requested one infrastructure retry for PR Gate run ${run.id}.`);
  return true;
}

module.exports = {recoverPRGate, membershipFailure};
