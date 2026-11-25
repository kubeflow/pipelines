// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
'use strict';

const {execFile} = require('node:child_process');
const {promisify} = require('node:util');
const {verifyCheckRuns} = require('./ci_check_runs');

const exec = promisify(execFile);
const repository = 'kubeflow/pipelines';

async function readRepository(path) {
  const {stdout} = await exec('gh', ['api', `repos/${repository}/${path}`], {
    timeout: 90_000, maxBuffer: 16 * 1024 * 1024,
  });
  return JSON.parse(stdout);
}

async function verifyStatuses(sha, replayPr, read) {
  // The combined-status endpoint returns the latest status for each context.
  const contexts = new Set();
  let total;
  for (let page = 1; page <= 10; page++) {
    const data = await read(`commits/${sha}/status?per_page=100&page=${page}`);
    if (data.sha !== sha || !Number.isSafeInteger(data.total_count) || data.total_count < 0 ||
        data.total_count >= 1000 || (total !== undefined && total !== data.total_count) ||
        !Array.isArray(data.statuses) || data.statuses.length > 100) {
      throw Error('Incomplete or changing landed commit statuses');
    }
    total = data.total_count;
    for (const status of data.statuses) {
      if (typeof status.context !== 'string' || !status.context || contexts.has(status.context)) {
        throw Error('Missing or duplicated landed commit status context');
      }
      contexts.add(status.context);
      const bookkeeping = status.state === 'pending' && (
        (status.context === 'tide' && status.target_url === 'https://oss.gprow.dev/tide') ||
        (status.context === 'merge-queue-admission' &&
          status.target_url === `https://github.com/${repository}/pull/${replayPr}`));
      if (status.state !== 'success' && !bookkeeping) {
        throw Error(`Landed commit status ${status.context} is ${status.state}`);
      }
    }
    if (contexts.size === total && contexts.has('ci-passed')) return;
    if (data.statuses.length < 100 || contexts.size >= total) break;
  }
  throw Error('Missing complete landed ci-passed status evidence');
}

async function checkLandedCommit(sha, replayPr, read = readRepository) {
  if (typeof sha !== 'string' || !/^[0-9a-f]{40}$/.test(sha) ||
      !Number.isSafeInteger(replayPr) || replayPr < 1) {
    throw Error('Expected the exact landed SHA and replay PR number');
  }
  const page = options => `per_page=${options.per_page}&page=${options.page}`;
  const get = async path => ({data: await read(path)});
  const github = {rest: {
    checks: {listForRef: options => get(`commits/${sha}/check-runs?filter=all&${page(options)}`)},
    actions: {
      listWorkflowRunsForRepo: options => get(`actions/runs?head_sha=${sha}&${page(options)}`),
      listJobsForWorkflowRunAttempt: options =>
        get(`actions/runs/${options.run_id}/attempts/${options.attempt_number}/jobs?${page(options)}`),
    },
  }};
  const [checks] = await Promise.all([
    verifyCheckRuns({github, owner: 'kubeflow', repo: 'pipelines', sha}),
    verifyStatuses(sha, replayPr, read),
  ]);
  if (checks.state !== 'success') throw Error(checks.reasons.join('; '));
}

if (require.main === module) {
  checkLandedCommit(process.argv[2], Number(process.argv[3])).catch(error => {
    console.error(error.message);
    process.exitCode = 1;
  });
}
module.exports = {checkLandedCommit};
